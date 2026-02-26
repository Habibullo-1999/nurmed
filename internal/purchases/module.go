package purchases

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
	ErrInvalidPurchasePayload       = errors.New("invalid purchase payload")
	ErrInvalidPurchaseReturnPayload = errors.New("invalid purchase return payload")
)

type Params struct {
	fx.In
	Logger       logger.ILogger
	PurchaseRepo interfaces.PurchaseRepo
}

type service struct {
	logger       logger.ILogger
	purchaseRepo interfaces.PurchaseRepo
}

type Service interface {
	ListAcquisitions(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrderResponse, error)
	CreateAcquisition(ctx context.Context, request structs.PurchaseCreateOrderRequest) (structs.PurchaseOrderResponse, error)
	ListRegistry(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrderResponse, error)
	ListReturns(ctx context.Context, request structs.PurchaseReturnFilter) ([]structs.PurchaseReturnResponse, error)
	CreateReturn(ctx context.Context, request structs.PurchaseCreateReturnRequest) (structs.PurchaseReturnResponse, error)
}

func New(p Params) Service {
	return &service{
		logger:       p.Logger,
		purchaseRepo: p.PurchaseRepo,
	}
}

func (s *service) ListAcquisitions(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrderResponse, error) {
	return s.listOrders(ctx, request)
}

func (s *service) CreateAcquisition(ctx context.Context, request structs.PurchaseCreateOrderRequest) (structs.PurchaseOrderResponse, error) {
	if request.CompanyID <= 0 || len(request.Items) == 0 {
		return structs.PurchaseOrderResponse{}, ErrInvalidPurchasePayload
	}

	request.SupplierName = strings.TrimSpace(request.SupplierName)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	request.DocumentNo = strings.TrimSpace(request.DocumentNo)
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))

	if request.Currency == "" {
		request.Currency = "UZS"
	}

	if request.Status == "" {
		request.Status = structs.PurchaseOrderStatusPosted
	}
	if !isValidOrderStatus(request.Status) {
		return structs.PurchaseOrderResponse{}, ErrInvalidPurchasePayload
	}

	if request.DocumentNo == "" {
		request.DocumentNo = generateDocumentNo()
	}

	purchasedAt := time.Now().UTC()
	if request.PurchasedAt != nil && !request.PurchasedAt.IsZero() {
		purchasedAt = request.PurchasedAt.UTC()
	}

	totalAmount := 0.0
	items := make([]structs.PurchaseOrderItem, 0, len(request.Items))
	for _, item := range request.Items {
		productName := strings.TrimSpace(item.ProductName)
		if productName == "" || item.Quantity <= 0 || item.Price < 0 {
			return structs.PurchaseOrderResponse{}, ErrInvalidPurchasePayload
		}

		amount := item.Quantity * item.Price
		totalAmount += amount
		items = append(items, structs.PurchaseOrderItem{
			ProductID:   item.ProductID,
			ProductName: productName,
			Quantity:    item.Quantity,
			Price:       item.Price,
			Amount:      amount,
		})
	}

	var createdBy *int64
	if request.CreatedBy > 0 {
		createdBy = &request.CreatedBy
	}

	createdOrder, createdItems, err := s.purchaseRepo.CreateOrder(ctx, structs.PurchaseOrder{
		CompanyID:    request.CompanyID,
		DocumentNo:   request.DocumentNo,
		SupplierName: request.SupplierName,
		Currency:     request.Currency,
		Status:       request.Status,
		TotalAmount:  totalAmount,
		ItemCount:    len(items),
		PurchasedAt:  purchasedAt,
		CreatedBy:    createdBy,
	}, items)
	if err != nil {
		return structs.PurchaseOrderResponse{}, err
	}

	responseItems := make([]structs.PurchaseOrderItemResponse, 0, len(createdItems))
	for _, item := range createdItems {
		responseItems = append(responseItems, mapOrderItemResponse(item))
	}

	return mapOrderResponse(createdOrder, responseItems), nil
}

func (s *service) ListRegistry(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrderResponse, error) {
	return s.listOrders(ctx, request)
}

func (s *service) ListReturns(ctx context.Context, request structs.PurchaseReturnFilter) ([]structs.PurchaseReturnResponse, error) {
	request.Validate()

	purchaseReturns, err := s.purchaseRepo.ListReturns(ctx, request)
	if err != nil {
		return nil, err
	}

	response := make([]structs.PurchaseReturnResponse, 0, len(purchaseReturns))
	for _, purchaseReturn := range purchaseReturns {
		response = append(response, mapReturnResponse(purchaseReturn))
	}

	return response, nil
}

func (s *service) CreateReturn(ctx context.Context, request structs.PurchaseCreateReturnRequest) (structs.PurchaseReturnResponse, error) {
	request.Reason = strings.TrimSpace(request.Reason)
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))

	if request.OrderID <= 0 || request.CompanyID <= 0 || request.TotalAmount <= 0 {
		return structs.PurchaseReturnResponse{}, ErrInvalidPurchaseReturnPayload
	}

	if request.Status == "" {
		request.Status = structs.PurchaseReturnStatusPosted
	}
	if !isValidReturnStatus(request.Status) {
		return structs.PurchaseReturnResponse{}, ErrInvalidPurchaseReturnPayload
	}

	returnedAt := time.Now().UTC()
	if request.ReturnedAt != nil && !request.ReturnedAt.IsZero() {
		returnedAt = request.ReturnedAt.UTC()
	}

	var createdBy *int64
	if request.CreatedBy > 0 {
		createdBy = &request.CreatedBy
	}

	createdReturn, err := s.purchaseRepo.CreateReturn(ctx, structs.PurchaseReturn{
		OrderID:     request.OrderID,
		CompanyID:   request.CompanyID,
		Reason:      request.Reason,
		Status:      request.Status,
		TotalAmount: request.TotalAmount,
		ReturnedAt:  returnedAt,
		CreatedBy:   createdBy,
	})
	if err != nil {
		return structs.PurchaseReturnResponse{}, err
	}

	return mapReturnResponse(createdReturn), nil
}

func (s *service) listOrders(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrderResponse, error) {
	request.Validate()

	orders, err := s.purchaseRepo.ListOrders(ctx, request)
	if err != nil {
		return nil, err
	}

	response := make([]structs.PurchaseOrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, mapOrderResponse(order, nil))
	}

	return response, nil
}

func generateDocumentNo() string {
	return fmt.Sprintf("PRC-%d-%s", time.Now().UTC().UnixNano(), randomDocumentSuffix())
}

func randomDocumentSuffix() string {
	bytes := make([]byte, 2)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%04x", time.Now().UTC().UnixNano()&0xffff)
	}
	return hex.EncodeToString(bytes)
}

func mapOrderResponse(order structs.PurchaseOrder, items []structs.PurchaseOrderItemResponse) structs.PurchaseOrderResponse {
	response := structs.PurchaseOrderResponse{
		ID:           order.ID,
		CompanyID:    order.CompanyID,
		DocumentNo:   order.DocumentNo,
		SupplierName: order.SupplierName,
		Currency:     order.Currency,
		Status:       order.Status,
		TotalAmount:  order.TotalAmount,
		ItemCount:    order.ItemCount,
		PurchasedAt:  order.PurchasedAt,
		CreatedBy:    order.CreatedBy,
		CreatedAt:    order.CreatedAt,
		UpdatedAt:    order.UpdatedAt,
	}

	if len(items) > 0 {
		response.Items = items
	}

	return response
}

func mapOrderItemResponse(item structs.PurchaseOrderItem) structs.PurchaseOrderItemResponse {
	return structs.PurchaseOrderItemResponse{
		ID:          item.ID,
		ProductID:   item.ProductID,
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		Price:       item.Price,
		Amount:      item.Amount,
	}
}

func mapReturnResponse(purchaseReturn structs.PurchaseReturn) structs.PurchaseReturnResponse {
	return structs.PurchaseReturnResponse{
		ID:          purchaseReturn.ID,
		OrderID:     purchaseReturn.OrderID,
		CompanyID:   purchaseReturn.CompanyID,
		Reason:      purchaseReturn.Reason,
		Status:      purchaseReturn.Status,
		TotalAmount: purchaseReturn.TotalAmount,
		ReturnedAt:  purchaseReturn.ReturnedAt,
		CreatedBy:   purchaseReturn.CreatedBy,
		CreatedAt:   purchaseReturn.CreatedAt,
		UpdatedAt:   purchaseReturn.UpdatedAt,
	}
}

func isValidOrderStatus(status string) bool {
	switch status {
	case structs.PurchaseOrderStatusDraft, structs.PurchaseOrderStatusPosted, structs.PurchaseOrderStatusCancelled:
		return true
	default:
		return false
	}
}

func isValidReturnStatus(status string) bool {
	switch status {
	case structs.PurchaseReturnStatusDraft, structs.PurchaseReturnStatusPosted, structs.PurchaseReturnStatusCancelled:
		return true
	default:
		return false
	}
}
