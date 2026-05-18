package structs

import "time"

const (
	CompanyStatusActive   = "active"
	CompanyStatusInactive = "inactive"
)

type Company struct {
	ID        int64
	Name      string
	Status    string
	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateCompanyRequest struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status"`
}

type CompanyResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UpdateCompanyRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type CompanyFilter struct {
	Name   string `form:"name"`
	Status string `form:"status"`
	Pagination
}

func (f *CompanyFilter) Validate() {
	f.Pagination.Validate()
}
