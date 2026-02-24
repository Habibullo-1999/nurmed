package structs

import (
	"strings"
	"time"
)

const (
	SalesChannelRealization = "realization"
	SalesChannelMobile      = "mobile"
	SalesChannelPOS         = "pos"

	SalesOrderStatusDraft     = "draft"
	SalesOrderStatusPosted    = "posted"
	SalesOrderStatusCancelled = "cancelled"

	SalesReturnStatusDraft     = "draft"
	SalesReturnStatusPosted    = "posted"
	SalesReturnStatusCancelled = "cancelled"
)

type SalesOrder struct {
	ID           int64
	CompanyID    int64
	Channel      string
	DocumentNo   string
	CustomerName string
	Currency     string
	Status       string
	TotalAmount  float64
	ItemCount    int
	SoldAt       time.Time
	CreatedBy    *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type SalesOrderItem struct {
	ID          int64
	OrderID     int64
	ProductID   *int64
	ProductName string
	Quantity    float64
	Price       float64
	Amount      float64
	CreatedAt   time.Time
}

type SalesItemRequest struct {
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName" binding:"required"`
	Quantity    float64 `json:"quantity" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
}

type SalesCreateOrderRequest struct {
	CompanyID    int64              `json:"companyId"`
	DocumentNo   string             `json:"documentNo"`
	CustomerName string             `json:"customerName"`
	Currency     string             `json:"currency"`
	Status       string             `json:"status"`
	SoldAt       *time.Time         `json:"soldAt"`
	Items        []SalesItemRequest `json:"items" binding:"required"`
	CreatedBy    int64              `json:"-"`
}

type SalesOrderItemResponse struct {
	ID          int64   `json:"id"`
	ProductID   *int64  `json:"productId,omitempty"`
	ProductName string  `json:"productName"`
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`
}

type SalesOrderResponse struct {
	ID           int64                    `json:"id"`
	CompanyID    int64                    `json:"companyID"`
	Channel      string                   `json:"channel"`
	DocumentNo   string                   `json:"documentNo"`
	CustomerName string                   `json:"customerName"`
	Currency     string                   `json:"currency"`
	Status       string                   `json:"status"`
	TotalAmount  float64                  `json:"totalAmount"`
	ItemCount    int                      `json:"itemCount"`
	SoldAt       time.Time                `json:"soldAt"`
	CreatedBy    *int64                   `json:"createdBy,omitempty"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
	Items        []SalesOrderItemResponse `json:"items,omitempty"`
}

type SalesOrderFilter struct {
	ID           int64  `form:"id" json:"id,omitempty"`
	CompanyID    int64  `form:"company_id" json:"companyID,omitempty"`
	DocumentNo   string `form:"document_no" json:"documentNo,omitempty"`
	CustomerName string `form:"customer_name" json:"customerName,omitempty"`
	Status       string `form:"status" json:"status,omitempty"`
	Pagination
}

func (f *SalesOrderFilter) Validate() {
	f.DocumentNo = strings.TrimSpace(f.DocumentNo)
	f.CustomerName = strings.TrimSpace(f.CustomerName)
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}

type SalesReturn struct {
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

type SalesCreateReturnRequest struct {
	OrderID     int64      `json:"orderId" binding:"required"`
	CompanyID   int64      `json:"companyId"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	TotalAmount float64    `json:"totalAmount" binding:"required"`
	ReturnedAt  *time.Time `json:"returnedAt"`
	CreatedBy   int64      `json:"-"`
}

type SalesReturnResponse struct {
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

type SalesReturnFilter struct {
	ID        int64  `form:"id" json:"id,omitempty"`
	OrderID   int64  `form:"order_id" json:"orderID,omitempty"`
	CompanyID int64  `form:"company_id" json:"companyID,omitempty"`
	Status    string `form:"status" json:"status,omitempty"`
	Pagination
}

func (f *SalesReturnFilter) Validate() {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}
