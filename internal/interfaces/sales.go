package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type SalesRepo interface {
	ListOrders(ctx context.Context, request structs.SalesOrderFilter, channel *string) ([]structs.SalesOrder, error)
	CreateOrder(ctx context.Context, order structs.SalesOrder, items []structs.SalesOrderItem) (structs.SalesOrder, []structs.SalesOrderItem, error)
	ListReturns(ctx context.Context, request structs.SalesReturnFilter) ([]structs.SalesReturn, error)
	CreateReturn(ctx context.Context, salesReturn structs.SalesReturn) (structs.SalesReturn, error)
}
