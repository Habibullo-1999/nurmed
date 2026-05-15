package warehouse

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/fx"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

var (
	ErrInvalidInventoryPayload = errors.New("invalid inventory payload")
	ErrInvalidTransferPayload  = errors.New("invalid transfer payload")
	ErrInvalidWriteoffPayload  = errors.New("invalid writeoff payload")
	ErrOrderAlreadyPosted      = errors.New("order already posted")
	ErrInsufficientStock       = errors.New("insufficient stock")
)

type Params struct {
	fx.In
	Logger        logger.ILogger
	WarehouseRepo interfaces.WarehouseRepo
}

type service struct {
	logger        logger.ILogger
	warehouseRepo interfaces.WarehouseRepo
}

type Service interface {
	ListWarehouses(ctx context.Context, companyID int64) ([]structs.WarehouseResponse, error)
	GetStock(ctx context.Context, filter structs.WarehouseStockFilter) ([]structs.WarehouseStockResponse, error)

	ListInventories(ctx context.Context, filter structs.InventoryOrderFilter) ([]structs.InventoryOrderResponse, error)
	CreateInventory(ctx context.Context, req structs.CreateInventoryOrderRequest) (structs.InventoryOrderResponse, error)
	PostInventory(ctx context.Context, id int64, companyID int64, updatedBy int64) (structs.InventoryOrderResponse, error)

	ListTransfers(ctx context.Context, filter structs.TransferOrderFilter) ([]structs.TransferOrderResponse, error)
	CreateTransfer(ctx context.Context, req structs.CreateTransferOrderRequest) (structs.TransferOrderResponse, error)
	PostTransfer(ctx context.Context, id int64, companyID int64) (structs.TransferOrderResponse, error)

	ListWriteoffs(ctx context.Context, filter structs.WriteoffOrderFilter) ([]structs.WriteoffOrderResponse, error)
	CreateWriteoff(ctx context.Context, req structs.CreateWriteoffOrderRequest) (structs.WriteoffOrderResponse, error)
	PostWriteoff(ctx context.Context, id int64, companyID int64) (structs.WriteoffOrderResponse, error)
}

func New(p Params) Service {
	return &service{
		logger:        p.Logger,
		warehouseRepo: p.WarehouseRepo,
	}
}

func (s *service) ListWarehouses(ctx context.Context, companyID int64) ([]structs.WarehouseResponse, error) {
	warehouses, err := s.warehouseRepo.ListWarehouses(ctx, companyID)
	if err != nil {
		return nil, err
	}

	result := make([]structs.WarehouseResponse, 0, len(warehouses))
	for _, w := range warehouses {
		result = append(result, mapWarehouseResponse(w))
	}
	return result, nil
}

func (s *service) GetStock(ctx context.Context, filter structs.WarehouseStockFilter) ([]structs.WarehouseStockResponse, error) {
	filter.Validate()
	return s.warehouseRepo.GetStock(ctx, filter)
}

// ---- Inventory ----

func (s *service) ListInventories(ctx context.Context, filter structs.InventoryOrderFilter) ([]structs.InventoryOrderResponse, error) {
	filter.Validate()
	return s.warehouseRepo.ListInventories(ctx, filter)
}

func (s *service) CreateInventory(ctx context.Context, req structs.CreateInventoryOrderRequest) (structs.InventoryOrderResponse, error) {
	if req.CompanyID <= 0 || req.WarehouseID <= 0 || len(req.Items) == 0 {
		return structs.InventoryOrderResponse{}, ErrInvalidInventoryPayload
	}

	req.Note = strings.TrimSpace(req.Note)
	req.DocumentNo = strings.TrimSpace(req.DocumentNo)
	if req.DocumentNo == "" {
		req.DocumentNo = generateDocumentNo("INV")
	}

	var createdBy *int64
	if req.CreatedBy > 0 {
		createdBy = &req.CreatedBy
	}

	items := make([]structs.InventoryOrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		productName := strings.TrimSpace(item.ProductName)
		if productName == "" {
			return structs.InventoryOrderResponse{}, ErrInvalidInventoryPayload
		}
		items = append(items, structs.InventoryOrderItem{
			ProductID:   item.ProductID,
			ProductName: productName,
			ExpectedQty: item.ExpectedQty,
			ActualQty:   item.ActualQty,
			CostPrice:   item.CostPrice,
		})
	}

	order, _, err := s.warehouseRepo.CreateInventory(ctx, structs.InventoryOrder{
		CompanyID:   req.CompanyID,
		WarehouseID: req.WarehouseID,
		DocumentNo:  req.DocumentNo,
		Status:      structs.InventoryStatusDraft,
		Note:        req.Note,
		CreatedBy:   createdBy,
	}, items)
	if err != nil {
		return structs.InventoryOrderResponse{}, err
	}

	return s.warehouseRepo.GetInventory(ctx, order.ID, req.CompanyID)
}

func (s *service) PostInventory(ctx context.Context, id int64, companyID int64, updatedBy int64) (structs.InventoryOrderResponse, error) {
	if id <= 0 || companyID <= 0 {
		return structs.InventoryOrderResponse{}, ErrInvalidInventoryPayload
	}

	var updatedByPtr *int64
	if updatedBy > 0 {
		updatedByPtr = &updatedBy
	}

	return s.warehouseRepo.PostInventory(ctx, id, companyID, updatedByPtr)
}

// ---- Transfer ----

func (s *service) ListTransfers(ctx context.Context, filter structs.TransferOrderFilter) ([]structs.TransferOrderResponse, error) {
	filter.Validate()
	return s.warehouseRepo.ListTransfers(ctx, filter)
}

func (s *service) CreateTransfer(ctx context.Context, req structs.CreateTransferOrderRequest) (structs.TransferOrderResponse, error) {
	if req.CompanyID <= 0 || req.FromWarehouseID <= 0 || req.ToWarehouseID <= 0 || len(req.Items) == 0 {
		return structs.TransferOrderResponse{}, ErrInvalidTransferPayload
	}
	if req.FromWarehouseID == req.ToWarehouseID {
		return structs.TransferOrderResponse{}, ErrInvalidTransferPayload
	}

	req.Note = strings.TrimSpace(req.Note)
	req.DocumentNo = strings.TrimSpace(req.DocumentNo)
	if req.DocumentNo == "" {
		req.DocumentNo = generateDocumentNo("TRF")
	}

	transferredAt := time.Now().UTC()
	if req.TransferredAt != nil && !req.TransferredAt.IsZero() {
		transferredAt = req.TransferredAt.UTC()
	}

	var createdBy *int64
	if req.CreatedBy > 0 {
		createdBy = &req.CreatedBy
	}

	items := make([]structs.TransferOrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		productName := strings.TrimSpace(item.ProductName)
		if productName == "" || item.Quantity <= 0 {
			return structs.TransferOrderResponse{}, ErrInvalidTransferPayload
		}
		items = append(items, structs.TransferOrderItem{
			ProductID:   item.ProductID,
			ProductName: productName,
			Quantity:    item.Quantity,
			CostPrice:   item.CostPrice,
		})
	}

	order, _, err := s.warehouseRepo.CreateTransfer(ctx, structs.TransferOrder{
		CompanyID:       req.CompanyID,
		DocumentNo:      req.DocumentNo,
		FromWarehouseID: req.FromWarehouseID,
		ToWarehouseID:   req.ToWarehouseID,
		Status:          structs.TransferStatusDraft,
		Note:            req.Note,
		TransferredAt:   transferredAt,
		CreatedBy:       createdBy,
	}, items)
	if err != nil {
		return structs.TransferOrderResponse{}, err
	}

	return s.warehouseRepo.GetTransfer(ctx, order.ID, req.CompanyID)
}

func (s *service) PostTransfer(ctx context.Context, id int64, companyID int64) (structs.TransferOrderResponse, error) {
	if id <= 0 || companyID <= 0 {
		return structs.TransferOrderResponse{}, ErrInvalidTransferPayload
	}
	return s.warehouseRepo.PostTransfer(ctx, id, companyID)
}

// ---- Writeoff ----

func (s *service) ListWriteoffs(ctx context.Context, filter structs.WriteoffOrderFilter) ([]structs.WriteoffOrderResponse, error) {
	filter.Validate()
	return s.warehouseRepo.ListWriteoffs(ctx, filter)
}

func (s *service) CreateWriteoff(ctx context.Context, req structs.CreateWriteoffOrderRequest) (structs.WriteoffOrderResponse, error) {
	if req.CompanyID <= 0 || req.WarehouseID <= 0 || len(req.Items) == 0 {
		return structs.WriteoffOrderResponse{}, ErrInvalidWriteoffPayload
	}

	req.Note = strings.TrimSpace(req.Note)
	req.ObjectName = strings.TrimSpace(req.ObjectName)
	req.CounterpartyName = strings.TrimSpace(req.CounterpartyName)
	req.DocumentNo = strings.TrimSpace(req.DocumentNo)
	if req.DocumentNo == "" {
		req.DocumentNo = generateDocumentNo("WOF")
	}

	writtenOffAt := time.Now().UTC()
	if req.WrittenOffAt != nil && !req.WrittenOffAt.IsZero() {
		writtenOffAt = req.WrittenOffAt.UTC()
	}

	var createdBy *int64
	if req.CreatedBy > 0 {
		createdBy = &req.CreatedBy
	}

	totalAmount := 0.0
	items := make([]structs.WriteoffOrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		productName := strings.TrimSpace(item.ProductName)
		if productName == "" || item.Quantity <= 0 {
			return structs.WriteoffOrderResponse{}, ErrInvalidWriteoffPayload
		}
		amount := item.Quantity * item.CostPrice
		totalAmount += amount
		items = append(items, structs.WriteoffOrderItem{
			ProductID:   item.ProductID,
			ProductName: productName,
			Quantity:    item.Quantity,
			CostPrice:   item.CostPrice,
			Amount:      amount,
		})
	}

	order, _, err := s.warehouseRepo.CreateWriteoff(ctx, structs.WriteoffOrder{
		CompanyID:        req.CompanyID,
		WarehouseID:      req.WarehouseID,
		DocumentNo:       req.DocumentNo,
		Status:           structs.WriteoffStatusDraft,
		TotalAmount:      totalAmount,
		ObjectName:       req.ObjectName,
		CounterpartyName: req.CounterpartyName,
		Note:             req.Note,
		WrittenOffAt:     writtenOffAt,
		CreatedBy:        createdBy,
	}, items)
	if err != nil {
		return structs.WriteoffOrderResponse{}, err
	}

	return s.warehouseRepo.GetWriteoff(ctx, order.ID, req.CompanyID)
}

func (s *service) PostWriteoff(ctx context.Context, id int64, companyID int64) (structs.WriteoffOrderResponse, error) {
	if id <= 0 || companyID <= 0 {
		return structs.WriteoffOrderResponse{}, ErrInvalidWriteoffPayload
	}
	return s.warehouseRepo.PostWriteoff(ctx, id, companyID)
}

// ---- Helpers ----

func mapWarehouseResponse(w structs.Warehouse) structs.WarehouseResponse {
	return structs.WarehouseResponse{
		ID:        w.ID,
		CompanyID: w.CompanyID,
		Name:      w.Name,
		Address:   w.Address,
		Status:    w.Status,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func generateDocumentNo(prefix string) string {
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixNano(), randomSuffix())
}

func randomSuffix() string {
	b := make([]byte, 2)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%04x", time.Now().UTC().UnixNano()&0xffff)
	}
	return hex.EncodeToString(b)
}
