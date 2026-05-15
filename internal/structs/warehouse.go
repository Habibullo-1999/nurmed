package structs

import (
	"strings"
	"time"
)

const (
	WarehouseStatusActive   = "active"
	WarehouseStatusInactive = "inactive"

	InventoryStatusDraft     = "draft"
	InventoryStatusPosted    = "posted"
	InventoryStatusCancelled = "cancelled"

	TransferStatusDraft     = "draft"
	TransferStatusPosted    = "posted"
	TransferStatusCancelled = "cancelled"

	WriteoffStatusDraft     = "draft"
	WriteoffStatusPosted    = "posted"
	WriteoffStatusCancelled = "cancelled"
)

// ---- Warehouse ----

type Warehouse struct {
	ID        int64
	CompanyID int64
	Name      string
	Address   string
	Status    string
	CreatedBy *int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WarehouseResponse struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"companyID"`
	Name      string    `json:"name"`
	Address   string    `json:"address,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ---- Stock ----

type WarehouseStockFilter struct {
	CompanyID   int64  `form:"company_id"`
	WarehouseID int64  `form:"warehouse_id"`
	Search      string `form:"search"`
	Pagination
}

func (f *WarehouseStockFilter) Validate() {
	f.Search = strings.TrimSpace(f.Search)
	f.Pagination.Validate()
}

type WarehouseStockResponse struct {
	ID             int64   `json:"id"`
	WarehouseID    int64   `json:"warehouseId"`
	WarehouseName  string  `json:"warehouseName"`
	ProductID      int64   `json:"productId"`
	ProductName    string  `json:"productName"`
	SKU            string  `json:"sku,omitempty"`
	Barcode        string  `json:"barcode,omitempty"`
	Unit           string  `json:"unit"`
	Quantity       float64 `json:"quantity"`
	CostPrice      float64 `json:"costPrice"`
	TotalCostPrice float64 `json:"totalCostPrice"`
}

// ---- Inventory ----

type InventoryOrder struct {
	ID             int64
	CompanyID      int64
	WarehouseID    int64
	DocumentNo     string
	Status         string
	SurplusDeficit float64
	Note           string
	CreatedBy      *int64
	UpdatedBy      *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type InventoryOrderItem struct {
	ID          int64
	OrderID     int64
	ProductID   *int64
	ProductName string
	ExpectedQty float64
	ActualQty   float64
	CostPrice   float64
	CreatedAt   time.Time
}

type InventoryOrderFilter struct {
	ID          int64      `form:"id"`
	CompanyID   int64      `form:"company_id"`
	WarehouseID int64      `form:"warehouse_id"`
	Status      string     `form:"status"`
	DateFrom    *time.Time `form:"date_from" time_format:"2006-01-02"`
	DateTo      *time.Time `form:"date_to" time_format:"2006-01-02"`
	Search      string     `form:"search"`
	Pagination
}

func (f *InventoryOrderFilter) Validate() {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Search = strings.TrimSpace(f.Search)
	f.Pagination.Validate()
}

type InventoryItemRequest struct {
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName" binding:"required"`
	ExpectedQty float64 `json:"expectedQty"`
	ActualQty   float64 `json:"actualQty"`
	CostPrice   float64 `json:"costPrice"`
}

type CreateInventoryOrderRequest struct {
	CompanyID   int64                  `json:"companyId"`
	WarehouseID int64                  `json:"warehouseId" binding:"required"`
	DocumentNo  string                 `json:"documentNo"`
	Note        string                 `json:"note"`
	Items       []InventoryItemRequest `json:"items" binding:"required"`
	CreatedBy   int64                  `json:"-"`
}

type InventoryOrderItemResponse struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName"`
	ExpectedQty float64 `json:"expectedQty"`
	ActualQty   float64 `json:"actualQty"`
	CostPrice   float64 `json:"costPrice"`
}

type InventoryOrderResponse struct {
	ID             int64                        `json:"id"`
	CompanyID      int64                        `json:"companyID"`
	WarehouseID    int64                        `json:"warehouseId"`
	WarehouseName  string                       `json:"warehouseName"`
	DocumentNo     string                       `json:"documentNo"`
	Status         string                       `json:"status"`
	SurplusDeficit float64                      `json:"surplusDeficit"`
	Note           string                       `json:"note,omitempty"`
	CreatedBy      *int64                       `json:"createdBy,omitempty"`
	CreatedByName  string                       `json:"createdByName,omitempty"`
	UpdatedBy      *int64                       `json:"updatedBy,omitempty"`
	UpdatedByName  string                       `json:"updatedByName,omitempty"`
	CreatedAt      time.Time                    `json:"createdAt"`
	UpdatedAt      time.Time                    `json:"updatedAt"`
	Items          []InventoryOrderItemResponse `json:"items,omitempty"`
}

// ---- Transfer ----

type TransferOrder struct {
	ID              int64
	CompanyID       int64
	DocumentNo      string
	FromWarehouseID int64
	ToWarehouseID   int64
	Status          string
	Note            string
	TransferredAt   time.Time
	ReceivedAt      *time.Time
	CreatedBy       *int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TransferOrderItem struct {
	ID          int64
	OrderID     int64
	ProductID   *int64
	ProductName string
	Quantity    float64
	CostPrice   float64
	CreatedAt   time.Time
}

type TransferOrderFilter struct {
	ID              int64      `form:"id"`
	CompanyID       int64      `form:"company_id"`
	FromWarehouseID int64      `form:"from_warehouse_id"`
	ToWarehouseID   int64      `form:"to_warehouse_id"`
	Status          string     `form:"status"`
	DateFrom        *time.Time `form:"date_from" time_format:"2006-01-02"`
	DateTo          *time.Time `form:"date_to" time_format:"2006-01-02"`
	Pagination
}

func (f *TransferOrderFilter) Validate() {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}

type TransferItemRequest struct {
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName" binding:"required"`
	Quantity    float64 `json:"quantity" binding:"required"`
	CostPrice   float64 `json:"costPrice"`
}

type CreateTransferOrderRequest struct {
	CompanyID       int64                 `json:"companyId"`
	FromWarehouseID int64                 `json:"fromWarehouseId" binding:"required"`
	ToWarehouseID   int64                 `json:"toWarehouseId" binding:"required"`
	DocumentNo      string                `json:"documentNo"`
	Note            string                `json:"note"`
	TransferredAt   *time.Time            `json:"transferredAt"`
	Items           []TransferItemRequest `json:"items" binding:"required"`
	CreatedBy       int64                 `json:"-"`
}

type TransferOrderItemResponse struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName"`
	Quantity    float64 `json:"quantity"`
	CostPrice   float64 `json:"costPrice"`
}

type TransferOrderResponse struct {
	ID                int64                       `json:"id"`
	CompanyID         int64                       `json:"companyID"`
	DocumentNo        string                      `json:"documentNo"`
	FromWarehouseID   int64                       `json:"fromWarehouseId"`
	FromWarehouseName string                      `json:"fromWarehouseName,omitempty"`
	ToWarehouseID     int64                       `json:"toWarehouseId"`
	ToWarehouseName   string                      `json:"toWarehouseName,omitempty"`
	Status            string                      `json:"status"`
	Note              string                      `json:"note,omitempty"`
	TransferredAt     time.Time                   `json:"transferredAt"`
	ReceivedAt        *time.Time                  `json:"receivedAt,omitempty"`
	CreatedBy         *int64                      `json:"createdBy,omitempty"`
	CreatedAt         time.Time                   `json:"createdAt"`
	UpdatedAt         time.Time                   `json:"updatedAt"`
	Items             []TransferOrderItemResponse `json:"items,omitempty"`
}

// ---- Writeoff ----

type WriteoffOrder struct {
	ID               int64
	CompanyID        int64
	WarehouseID      int64
	DocumentNo       string
	Status           string
	TotalAmount      float64
	ObjectName       string
	CounterpartyName string
	Note             string
	WrittenOffAt     time.Time
	CreatedBy        *int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type WriteoffOrderItem struct {
	ID          int64
	OrderID     int64
	ProductID   *int64
	ProductName string
	Quantity    float64
	CostPrice   float64
	Amount      float64
	CreatedAt   time.Time
}

type WriteoffOrderFilter struct {
	ID          int64      `form:"id"`
	CompanyID   int64      `form:"company_id"`
	WarehouseID int64      `form:"warehouse_id"`
	Status      string     `form:"status"`
	DateFrom    *time.Time `form:"date_from" time_format:"2006-01-02"`
	DateTo      *time.Time `form:"date_to" time_format:"2006-01-02"`
	Pagination
}

func (f *WriteoffOrderFilter) Validate() {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}

type WriteoffItemRequest struct {
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName" binding:"required"`
	Quantity    float64 `json:"quantity" binding:"required"`
	CostPrice   float64 `json:"costPrice"`
}

type CreateWriteoffOrderRequest struct {
	CompanyID        int64                 `json:"companyId"`
	WarehouseID      int64                 `json:"warehouseId" binding:"required"`
	DocumentNo       string                `json:"documentNo"`
	ObjectName       string                `json:"objectName"`
	CounterpartyName string                `json:"counterpartyName"`
	Note             string                `json:"note"`
	WrittenOffAt     *time.Time            `json:"writtenOffAt"`
	Items            []WriteoffItemRequest `json:"items" binding:"required"`
	CreatedBy        int64                 `json:"-"`
}

type WriteoffOrderItemResponse struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName"`
	Quantity    float64 `json:"quantity"`
	CostPrice   float64 `json:"costPrice"`
	Amount      float64 `json:"amount"`
}

type WriteoffOrderResponse struct {
	ID               int64                       `json:"id"`
	CompanyID        int64                       `json:"companyID"`
	WarehouseID      int64                       `json:"warehouseId"`
	WarehouseName    string                      `json:"warehouseName,omitempty"`
	DocumentNo       string                      `json:"documentNo"`
	Status           string                      `json:"status"`
	TotalAmount      float64                     `json:"totalAmount"`
	ObjectName       string                      `json:"objectName,omitempty"`
	CounterpartyName string                      `json:"counterpartyName,omitempty"`
	Note             string                      `json:"note,omitempty"`
	WrittenOffAt     time.Time                   `json:"writtenOffAt"`
	CreatedBy        *int64                      `json:"createdBy,omitempty"`
	CreatedByName    string                      `json:"createdByName,omitempty"`
	CreatedAt        time.Time                   `json:"createdAt"`
	UpdatedAt        time.Time                   `json:"updatedAt"`
	Items            []WriteoffOrderItemResponse `json:"items,omitempty"`
}
