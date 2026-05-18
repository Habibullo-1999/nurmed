package interfaces

import (
	"context"

	"nurmed/internal/structs"
)

type CompanyRepo interface {
	CreateCompany(ctx context.Context, company structs.Company) (structs.Company, error)
	ListCompanies(ctx context.Context, filter structs.CompanyFilter) ([]structs.Company, error)
	UpdateCompany(ctx context.Context, id int64, req structs.UpdateCompanyRequest) (structs.Company, error)
	DeleteCompany(ctx context.Context, id int64) error
}
