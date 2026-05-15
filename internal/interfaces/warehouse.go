package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type WarehouseRepo interface {
	ListWarehouses(ctx context.Context, companyID int64) ([]structs.Warehouse, error)
	GetStock(ctx context.Context, filter structs.WarehouseStockFilter) ([]structs.WarehouseStockResponse, error)

	ListInventories(ctx context.Context, filter structs.InventoryOrderFilter) ([]structs.InventoryOrderResponse, error)
	CreateInventory(ctx context.Context, order structs.InventoryOrder, items []structs.InventoryOrderItem) (structs.InventoryOrder, []structs.InventoryOrderItem, error)
	GetInventory(ctx context.Context, id int64, companyID int64) (structs.InventoryOrderResponse, error)
	PostInventory(ctx context.Context, id int64, companyID int64, updatedBy *int64) (structs.InventoryOrderResponse, error)

	ListTransfers(ctx context.Context, filter structs.TransferOrderFilter) ([]structs.TransferOrderResponse, error)
	CreateTransfer(ctx context.Context, order structs.TransferOrder, items []structs.TransferOrderItem) (structs.TransferOrder, []structs.TransferOrderItem, error)
	GetTransfer(ctx context.Context, id int64, companyID int64) (structs.TransferOrderResponse, error)
	PostTransfer(ctx context.Context, id int64, companyID int64) (structs.TransferOrderResponse, error)

	ListWriteoffs(ctx context.Context, filter structs.WriteoffOrderFilter) ([]structs.WriteoffOrderResponse, error)
	CreateWriteoff(ctx context.Context, order structs.WriteoffOrder, items []structs.WriteoffOrderItem) (structs.WriteoffOrder, []structs.WriteoffOrderItem, error)
	GetWriteoff(ctx context.Context, id int64, companyID int64) (structs.WriteoffOrderResponse, error)
	PostWriteoff(ctx context.Context, id int64, companyID int64) (structs.WriteoffOrderResponse, error)
}
