package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type PurchaseRepo interface {
	ListOrders(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrder, error)
	CreateOrder(ctx context.Context, order structs.PurchaseOrder, items []structs.PurchaseOrderItem) (structs.PurchaseOrder, []structs.PurchaseOrderItem, error)
	ListReturns(ctx context.Context, request structs.PurchaseReturnFilter) ([]structs.PurchaseReturn, error)
	CreateReturn(ctx context.Context, purchaseReturn structs.PurchaseReturn) (structs.PurchaseReturn, error)
}
