package sales

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
	ErrInvalidSalesPayload  = errors.New("invalid sales payload")
	ErrInvalidReturnPayload = errors.New("invalid return payload")
)

type Params struct {
	fx.In
	Logger    logger.ILogger
	SalesRepo interfaces.SalesRepo
}

type service struct {
	logger    logger.ILogger
	salesRepo interfaces.SalesRepo
}

type Service interface {
	ListRealizations(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error)
	CreateRealization(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error)
	ListRegistry(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error)
	ListMobile(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error)
	CreateMobile(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error)
	ListPOS(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error)
	CreatePOS(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error)
	ListReturns(ctx context.Context, request structs.SalesReturnFilter) ([]structs.SalesReturnResponse, error)
	CreateReturn(ctx context.Context, request structs.SalesCreateReturnRequest) (structs.SalesReturnResponse, error)
}

func New(p Params) Service {
	return &service{
		logger:    p.Logger,
		salesRepo: p.SalesRepo,
	}
}

func (s *service) ListRealizations(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error) {
	return s.listOrdersByChannel(ctx, request, structs.SalesChannelRealization)
}

func (s *service) CreateRealization(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error) {
	return s.createOrderByChannel(ctx, request, structs.SalesChannelRealization)
}

func (s *service) ListRegistry(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error) {
	request.Validate()

	orders, err := s.salesRepo.ListOrders(ctx, request, nil)
	if err != nil {
		return nil, err
	}

	response := make([]structs.SalesOrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, mapOrderResponse(order, nil))
	}

	return response, nil
}

func (s *service) ListMobile(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error) {
	return s.listOrdersByChannel(ctx, request, structs.SalesChannelMobile)
}

func (s *service) CreateMobile(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error) {
	return s.createOrderByChannel(ctx, request, structs.SalesChannelMobile)
}

func (s *service) ListPOS(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error) {
	return s.listOrdersByChannel(ctx, request, structs.SalesChannelPOS)
}

func (s *service) CreatePOS(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error) {
	return s.createOrderByChannel(ctx, request, structs.SalesChannelPOS)
}

func (s *service) ListReturns(ctx context.Context, request structs.SalesReturnFilter) ([]structs.SalesReturnResponse, error) {
	request.Validate()

	salesReturns, err := s.salesRepo.ListReturns(ctx, request)
	if err != nil {
		return nil, err
	}

	response := make([]structs.SalesReturnResponse, 0, len(salesReturns))
	for _, salesReturn := range salesReturns {
		response = append(response, mapReturnResponse(salesReturn))
	}

	return response, nil
}

func (s *service) CreateReturn(ctx context.Context, request structs.SalesCreateReturnRequest) (structs.SalesReturnResponse, error) {
	request.Reason = strings.TrimSpace(request.Reason)
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))

	if request.OrderID <= 0 || request.CompanyID <= 0 || request.TotalAmount <= 0 {
		return structs.SalesReturnResponse{}, ErrInvalidReturnPayload
	}

	if request.Status == "" {
		request.Status = structs.SalesReturnStatusPosted
	}
	if !isValidReturnStatus(request.Status) {
		return structs.SalesReturnResponse{}, ErrInvalidReturnPayload
	}

	returnedAt := time.Now().UTC()
	if request.ReturnedAt != nil && !request.ReturnedAt.IsZero() {
		returnedAt = request.ReturnedAt.UTC()
	}

	var createdBy *int64
	if request.CreatedBy > 0 {
		createdBy = &request.CreatedBy
	}

	createdReturn, err := s.salesRepo.CreateReturn(ctx, structs.SalesReturn{
		OrderID:     request.OrderID,
		CompanyID:   request.CompanyID,
		Reason:      request.Reason,
		Status:      request.Status,
		TotalAmount: request.TotalAmount,
		ReturnedAt:  returnedAt,
		CreatedBy:   createdBy,
	})
	if err != nil {
		return structs.SalesReturnResponse{}, err
	}

	return mapReturnResponse(createdReturn), nil
}

func (s *service) listOrdersByChannel(ctx context.Context, request structs.SalesOrderFilter, channel string) ([]structs.SalesOrderResponse, error) {
	request.Validate()

	orders, err := s.salesRepo.ListOrders(ctx, request, &channel)
	if err != nil {
		return nil, err
	}

	response := make([]structs.SalesOrderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, mapOrderResponse(order, nil))
	}

	return response, nil
}

func (s *service) createOrderByChannel(ctx context.Context, request structs.SalesCreateOrderRequest, channel string) (structs.SalesOrderResponse, error) {
	if request.CompanyID <= 0 || len(request.Items) == 0 {
		return structs.SalesOrderResponse{}, ErrInvalidSalesPayload
	}

	request.CustomerName = strings.TrimSpace(request.CustomerName)
	request.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	request.DocumentNo = strings.TrimSpace(request.DocumentNo)
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))

	if request.Currency == "" {
		request.Currency = "UZS"
	}

	if request.Status == "" {
		request.Status = structs.SalesOrderStatusPosted
	}
	if !isValidOrderStatus(request.Status) {
		return structs.SalesOrderResponse{}, ErrInvalidSalesPayload
	}

	if request.DocumentNo == "" {
		request.DocumentNo = generateDocumentNo(channel)
	}

	soldAt := time.Now().UTC()
	if request.SoldAt != nil && !request.SoldAt.IsZero() {
		soldAt = request.SoldAt.UTC()
	}

	totalAmount := 0.0
	items := make([]structs.SalesOrderItem, 0, len(request.Items))
	for _, item := range request.Items {
		productName := strings.TrimSpace(item.ProductName)
		if productName == "" || item.Quantity <= 0 || item.Price < 0 {
			return structs.SalesOrderResponse{}, ErrInvalidSalesPayload
		}

		amount := item.Quantity * item.Price
		totalAmount += amount
		items = append(items, structs.SalesOrderItem{
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

	createdOrder, createdItems, err := s.salesRepo.CreateOrder(ctx, structs.SalesOrder{
		CompanyID:    request.CompanyID,
		Channel:      channel,
		DocumentNo:   request.DocumentNo,
		CustomerName: request.CustomerName,
		Currency:     request.Currency,
		Status:       request.Status,
		TotalAmount:  totalAmount,
		ItemCount:    len(items),
		SoldAt:       soldAt,
		CreatedBy:    createdBy,
	}, items)
	if err != nil {
		return structs.SalesOrderResponse{}, err
	}

	responseItems := make([]structs.SalesOrderItemResponse, 0, len(createdItems))
	for _, item := range createdItems {
		responseItems = append(responseItems, mapOrderItemResponse(item))
	}

	return mapOrderResponse(createdOrder, responseItems), nil
}

func generateDocumentNo(channel string) string {
	prefix := "SLS"
	switch channel {
	case structs.SalesChannelRealization:
		prefix = "RLZ"
	case structs.SalesChannelMobile:
		prefix = "MBL"
	case structs.SalesChannelPOS:
		prefix = "POS"
	}

	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UTC().UnixNano(), randomDocumentSuffix())
}

func randomDocumentSuffix() string {
	bytes := make([]byte, 2)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%04x", time.Now().UTC().UnixNano()&0xffff)
	}
	return hex.EncodeToString(bytes)
}

func mapOrderResponse(order structs.SalesOrder, items []structs.SalesOrderItemResponse) structs.SalesOrderResponse {
	response := structs.SalesOrderResponse{
		ID:           order.ID,
		CompanyID:    order.CompanyID,
		Channel:      order.Channel,
		DocumentNo:   order.DocumentNo,
		CustomerName: order.CustomerName,
		Currency:     order.Currency,
		Status:       order.Status,
		TotalAmount:  order.TotalAmount,
		ItemCount:    order.ItemCount,
		SoldAt:       order.SoldAt,
		CreatedBy:    order.CreatedBy,
		CreatedAt:    order.CreatedAt,
		UpdatedAt:    order.UpdatedAt,
	}

	if len(items) > 0 {
		response.Items = items
	}

	return response
}

func mapOrderItemResponse(item structs.SalesOrderItem) structs.SalesOrderItemResponse {
	return structs.SalesOrderItemResponse{
		ID:          item.ID,
		ProductID:   item.ProductID,
		ProductName: item.ProductName,
		Quantity:    item.Quantity,
		Price:       item.Price,
		Amount:      item.Amount,
	}
}

func mapReturnResponse(salesReturn structs.SalesReturn) structs.SalesReturnResponse {
	return structs.SalesReturnResponse{
		ID:          salesReturn.ID,
		OrderID:     salesReturn.OrderID,
		CompanyID:   salesReturn.CompanyID,
		Reason:      salesReturn.Reason,
		Status:      salesReturn.Status,
		TotalAmount: salesReturn.TotalAmount,
		ReturnedAt:  salesReturn.ReturnedAt,
		CreatedBy:   salesReturn.CreatedBy,
		CreatedAt:   salesReturn.CreatedAt,
		UpdatedAt:   salesReturn.UpdatedAt,
	}
}

func isValidOrderStatus(status string) bool {
	switch status {
	case structs.SalesOrderStatusDraft, structs.SalesOrderStatusPosted, structs.SalesOrderStatusCancelled:
		return true
	default:
		return false
	}
}

func isValidReturnStatus(status string) bool {
	switch status {
	case structs.SalesReturnStatusDraft, structs.SalesReturnStatusPosted, structs.SalesReturnStatusCancelled:
		return true
	default:
		return false
	}
}
