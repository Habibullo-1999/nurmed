package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type ProductRepo interface {
	ListProducts(ctx context.Context, request structs.ProductFilter) ([]structs.Product, error)
	CreateProduct(ctx context.Context, product structs.Product) (structs.Product, error)
}
