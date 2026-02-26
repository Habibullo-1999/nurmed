package structs

import (
	"strings"
	"time"
)

const (
	PurchaseOrderStatusDraft     = "draft"
	PurchaseOrderStatusPosted    = "posted"
	PurchaseOrderStatusCancelled = "cancelled"

	PurchaseReturnStatusDraft     = "draft"
	PurchaseReturnStatusPosted    = "posted"
	PurchaseReturnStatusCancelled = "cancelled"
)

type PurchaseOrder struct {
	ID           int64
	CompanyID    int64
	DocumentNo   string
	SupplierName string
	Currency     string
	Status       string
	TotalAmount  float64
	ItemCount    int
	PurchasedAt  time.Time
	CreatedBy    *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PurchaseOrderItem struct {
	ID          int64
	OrderID     int64
	ProductID   *int64
	ProductName string
	Quantity    float64
	Price       float64
	Amount      float64
	CreatedAt   time.Time
}

type PurchaseItemRequest struct {
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName" binding:"required"`
	Quantity    float64 `json:"quantity" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
}

type PurchaseCreateOrderRequest struct {
	CompanyID    int64                 `json:"companyId"`
	DocumentNo   string                `json:"documentNo"`
	SupplierName string                `json:"supplierName"`
	Currency     string                `json:"currency"`
	Status       string                `json:"status"`
	PurchasedAt  *time.Time            `json:"purchasedAt"`
	Items        []PurchaseItemRequest `json:"items" binding:"required"`
	CreatedBy    int64                 `json:"-"`
}

type PurchaseOrderItemResponse struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
}

type PurchaseOrderResponse struct {
	ID           int64                       `json:"id"`
	CompanyID    int64                       `json:"companyID"`
	DocumentNo   string                      `json:"documentNo"`
	SupplierName string                      `json:"supplierName"`
	Currency     string                      `json:"currency"`
	Status       string                      `json:"status"`
	TotalAmount  float64                     `json:"totalAmount"`
	ItemCount    int                         `json:"itemCount"`
	PurchasedAt  time.Time                   `json:"purchasedAt"`
	CreatedBy    *int64                      `json:"createdBy,omitempty"`
	CreatedAt    time.Time                   `json:"createdAt"`
	UpdatedAt    time.Time                   `json:"updatedAt"`
	Items        []PurchaseOrderItemResponse `json:"items,omitempty"`
}

type PurchaseOrderFilter struct {
	ID           int64  `form:"id" json:"id,omitempty"`
	CompanyID    int64  `form:"company_id" json:"companyID,omitempty"`
	DocumentNo   string `form:"document_no" json:"documentNo,omitempty"`
	SupplierName string `form:"supplier_name" json:"supplierName,omitempty"`
	Status       string `form:"status" json:"status,omitempty"`
	Pagination
}

func (f *PurchaseOrderFilter) Validate() {
	f.DocumentNo = strings.TrimSpace(f.DocumentNo)
	f.SupplierName = strings.TrimSpace(f.SupplierName)
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}

type PurchaseReturn struct {
	ID          int64
	OrderID     int64
	CompanyID   int64
	Reason      string
	Status      string
	TotalAmount float64
	ReturnedAt  time.Time
	CreatedBy   *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PurchaseCreateReturnRequest struct {
	OrderID     int64      `json:"orderId" binding:"required"`
	CompanyID   int64      `json:"companyId"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	TotalAmount float64    `json:"totalAmount" binding:"required"`
	ReturnedAt  *time.Time `json:"returnedAt"`
	CreatedBy   int64      `json:"-"`
}

type PurchaseReturnResponse struct {
	ID          int64     `json:"id"`
	OrderID     int64     `json:"orderId"`
	CompanyID   int64     `json:"companyID"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"totalAmount"`
	ReturnedAt  time.Time `json:"returnedAt"`
	CreatedBy   *int64    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PurchaseReturnFilter struct {
	ID        int64  `form:"id" json:"id,omitempty"`
	OrderID   int64  `form:"order_id" json:"orderID,omitempty"`
	CompanyID int64  `form:"company_id" json:"companyID,omitempty"`
	Status    string `form:"status" json:"status,omitempty"`
	Pagination
}

func (f *PurchaseReturnFilter) Validate() {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}
