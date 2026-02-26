package structs

import (
	"strings"
	"time"
)

const (
	ProductStatusActive   = "active"
	ProductStatusInactive = "inactive"
)

type Product struct {
	ID            int64
	CompanyID     int64
	Name          string
	SKU           string
	Barcode       string
	Unit          string
	PurchasePrice float64
	SalePrice     float64
	Status        string
	CreatedBy     *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ProductFilter struct {
	ID        int64  `form:"id" json:"id,omitempty"`
	CompanyID int64  `form:"company_id" json:"companyID,omitempty"`
	Name      string `form:"name" json:"name,omitempty"`
	SKU       string `form:"sku" json:"sku,omitempty"`
	Barcode   string `form:"barcode" json:"barcode,omitempty"`
	Status    string `form:"status" json:"status,omitempty"`
	Pagination
}

func (f *ProductFilter) Validate() {
	f.Name = strings.TrimSpace(f.Name)
	f.SKU = strings.TrimSpace(f.SKU)
	f.Barcode = strings.TrimSpace(f.Barcode)
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Pagination.Validate()
}

type CreateProductRequest struct {
	CompanyID     int64   `json:"companyId"`
	Name          string  `json:"name" binding:"required"`
	SKU           string  `json:"sku"`
	Barcode       string  `json:"barcode"`
	Unit          string  `json:"unit"`
	PurchasePrice float64 `json:"purchasePrice"`
	SalePrice     float64 `json:"salePrice"`
	Status        string  `json:"status"`
	CreatedBy     int64   `json:"-"`
}

type ProductResponse struct {
	ID            int64     `json:"id"`
	CompanyID     int64     `json:"companyID"`
	Name          string    `json:"name"`
	SKU           string    `json:"sku,omitempty"`
	Barcode       string    `json:"barcode,omitempty"`
	Unit          string    `json:"unit"`
	PurchasePrice float64   `json:"purchasePrice"`
	SalePrice     float64   `json:"salePrice"`
	Status        string    `json:"status"`
	CreatedBy     *int64    `json:"createdBy,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
