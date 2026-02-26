package products

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/fx"

	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

var (
	ErrInvalidProductPayload = errors.New("invalid product payload")
)

type Params struct {
	fx.In
	Logger      logger.ILogger
	ProductRepo interfaces.ProductRepo
}

type service struct {
	logger      logger.ILogger
	productRepo interfaces.ProductRepo
}

type Service interface {
	ListProducts(ctx context.Context, request structs.ProductFilter) ([]structs.ProductResponse, error)
	CreateProduct(ctx context.Context, request structs.CreateProductRequest) (structs.ProductResponse, error)
}

func New(p Params) Service {
	return &service{
		logger:      p.Logger,
		productRepo: p.ProductRepo,
	}
}

func (s *service) ListProducts(ctx context.Context, request structs.ProductFilter) ([]structs.ProductResponse, error) {
	request.Validate()

	products, err := s.productRepo.ListProducts(ctx, request)
	if err != nil {
		return nil, err
	}

	response := make([]structs.ProductResponse, 0, len(products))
	for _, product := range products {
		response = append(response, mapProductResponse(product))
	}

	return response, nil
}

func (s *service) CreateProduct(ctx context.Context, request structs.CreateProductRequest) (structs.ProductResponse, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.SKU = strings.TrimSpace(request.SKU)
	request.Barcode = strings.TrimSpace(request.Barcode)
	request.Unit = strings.TrimSpace(request.Unit)
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))

	if request.CompanyID <= 0 || request.Name == "" || request.PurchasePrice < 0 || request.SalePrice < 0 {
		return structs.ProductResponse{}, ErrInvalidProductPayload
	}

	if request.Unit == "" {
		request.Unit = "pcs"
	}

	if request.Status == "" {
		request.Status = structs.ProductStatusActive
	}
	if !isValidProductStatus(request.Status) {
		return structs.ProductResponse{}, ErrInvalidProductPayload
	}

	var createdBy *int64
	if request.CreatedBy > 0 {
		createdBy = &request.CreatedBy
	}

	product, err := s.productRepo.CreateProduct(ctx, structs.Product{
		CompanyID:     request.CompanyID,
		Name:          request.Name,
		SKU:           request.SKU,
		Barcode:       request.Barcode,
		Unit:          request.Unit,
		PurchasePrice: request.PurchasePrice,
		SalePrice:     request.SalePrice,
		Status:        request.Status,
		CreatedBy:     createdBy,
	})
	if err != nil {
		return structs.ProductResponse{}, err
	}

	return mapProductResponse(product), nil
}

func mapProductResponse(product structs.Product) structs.ProductResponse {
	return structs.ProductResponse{
		ID:            product.ID,
		CompanyID:     product.CompanyID,
		Name:          product.Name,
		SKU:           product.SKU,
		Barcode:       product.Barcode,
		Unit:          product.Unit,
		PurchasePrice: product.PurchasePrice,
		SalePrice:     product.SalePrice,
		Status:        product.Status,
		CreatedBy:     product.CreatedBy,
		CreatedAt:     product.CreatedAt,
		UpdatedAt:     product.UpdatedAt,
	}
}

func isValidProductStatus(status string) bool {
	switch status {
	case structs.ProductStatusActive, structs.ProductStatusInactive:
		return true
	default:
		return false
	}
}
