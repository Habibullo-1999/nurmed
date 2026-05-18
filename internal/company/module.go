package company

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/fx"

	intauth "nurmed/internal/auth"
	"nurmed/internal/interfaces"
	"nurmed/internal/structs"
	"nurmed/pkg/ctxutil"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

var (
	ErrInvalidCompanyPayload = errors.New("invalid company payload")
	ErrNotFound              = errors.New("not found")
)

type Params struct {
	fx.In
	Logger      logger.ILogger
	CompanyRepo interfaces.CompanyRepo
	AuthSvs     intauth.Service
}

type service struct {
	logger      logger.ILogger
	companyRepo interfaces.CompanyRepo
	authSvs     intauth.Service
}

type Service interface {
	CreateCompany(ctx context.Context, req structs.CreateCompanyRequest) (structs.CompanyResponse, error)
	GetCompanies(ctx context.Context, filter structs.CompanyFilter) ([]structs.CompanyResponse, error)
	UpdateCompany(ctx context.Context, id int64, req structs.UpdateCompanyRequest) (structs.CompanyResponse, error)
	DeleteCompany(ctx context.Context, id int64) error
}

func New(p Params) Service {
	return &service{
		logger:      p.Logger,
		companyRepo: p.CompanyRepo,
		authSvs:     p.AuthSvs,
	}
}

func (s *service) CreateCompany(ctx context.Context, req structs.CreateCompanyRequest) (structs.CompanyResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return structs.CompanyResponse{}, ErrInvalidCompanyPayload
	}

	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if req.Status == "" {
		req.Status = structs.CompanyStatusActive
	}
	if req.Status != structs.CompanyStatusActive && req.Status != structs.CompanyStatusInactive {
		return structs.CompanyResponse{}, ErrInvalidCompanyPayload
	}

	company, err := s.companyRepo.CreateCompany(ctx, structs.Company{
		Name:   req.Name,
		Status: req.Status,
	})
	if err != nil {
		return structs.CompanyResponse{}, err
	}

	resourceID := fmt.Sprintf("%d", company.ID)
	s.authSvs.Audit(ctx, structs.AuditLog{
		UserID:     ctxutil.ActorID(ctx),
		Action:     "company.create",
		Module:     "admin",
		Resource:   "company",
		ResourceID: &resourceID,
		Meta:       map[string]interface{}{"name": company.Name, "status": company.Status},
	})

	return toResponse(company), nil
}

func (s *service) GetCompanies(ctx context.Context, filter structs.CompanyFilter) ([]structs.CompanyResponse, error) {
	filter.Validate()

	companies, err := s.companyRepo.ListCompanies(ctx, filter)
	if err != nil {
		return nil, err
	}

	result := make([]structs.CompanyResponse, 0, len(companies))
	for _, c := range companies {
		result = append(result, toResponse(c))
	}
	return result, nil
}

func (s *service) UpdateCompany(ctx context.Context, id int64, req structs.UpdateCompanyRequest) (structs.CompanyResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if req.Status != "" && req.Status != structs.CompanyStatusActive && req.Status != structs.CompanyStatusInactive {
		return structs.CompanyResponse{}, ErrInvalidCompanyPayload
	}

	company, err := s.companyRepo.UpdateCompany(ctx, id, req)
	if err != nil {
		return structs.CompanyResponse{}, err
	}

	resourceID := fmt.Sprintf("%d", id)
	meta := map[string]interface{}{}
	if req.Name != "" {
		meta["name"] = req.Name
	}
	if req.Status != "" {
		meta["status"] = req.Status
	}
	s.authSvs.Audit(ctx, structs.AuditLog{
		UserID:     ctxutil.ActorID(ctx),
		Action:     "company.update",
		Module:     "admin",
		Resource:   "company",
		ResourceID: &resourceID,
		Meta:       meta,
	})

	return toResponse(company), nil
}

func (s *service) DeleteCompany(ctx context.Context, id int64) error {
	if err := s.companyRepo.DeleteCompany(ctx, id); err != nil {
		return err
	}

	resourceID := fmt.Sprintf("%d", id)
	s.authSvs.Audit(ctx, structs.AuditLog{
		UserID:     ctxutil.ActorID(ctx),
		Action:     "company.delete",
		Module:     "admin",
		Resource:   "company",
		ResourceID: &resourceID,
		Meta:       map[string]interface{}{},
	})

	return nil
}

func toResponse(c structs.Company) structs.CompanyResponse {
	return structs.CompanyResponse{
		ID:        c.ID,
		Name:      c.Name,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
