package company

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"nurmed/internal/company"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger     logger.ILogger
	CompanySvs company.Service
}

type handler struct {
	logger     logger.ILogger
	companySvs company.Service
}

type Handler interface {
	CreateCompany(c *gin.Context)
	GetCompanies(c *gin.Context)
	UpdateCompany(c *gin.Context)
	DeleteCompany(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:     p.Logger,
		companySvs: p.CompanySvs,
	}
}

func (h *handler) CreateCompany(c *gin.Context) {
	var (
		request  structs.CreateCompanyRequest
		response structs.Response
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/company CreateCompany bind failed", zap.Error(err))
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	result, err := h.companySvs.CreateCompany(c.Request.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, company.ErrInvalidCompanyPayload):
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/company CreateCompany failed", zap.Error(err))
			response = responses.InternalErr
		}
		c.JSON(response.Code, &response)
		return
	}

	response = responses.Success
	response.Payload = result
	c.JSON(response.Code, &response)
}

func (h *handler) UpdateCompany(c *gin.Context) {
	var (
		request  structs.UpdateCompanyRequest
		response structs.Response
	)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/company UpdateCompany bind failed", zap.Error(err))
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	result, err := h.companySvs.UpdateCompany(c.Request.Context(), id, request)
	if err != nil {
		switch {
		case errors.Is(err, company.ErrInvalidCompanyPayload):
			response = responses.BadRequest
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.NotFound
		default:
			h.logger.Error(c.Request.Context(), "handlers/company UpdateCompany failed", zap.Error(err))
			response = responses.InternalErr
		}
		c.JSON(response.Code, &response)
		return
	}

	response = responses.Success
	response.Payload = result
	c.JSON(response.Code, &response)
}

func (h *handler) DeleteCompany(c *gin.Context) {
	var response structs.Response

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	if err := h.companySvs.DeleteCompany(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.NotFound
		default:
			h.logger.Error(c.Request.Context(), "handlers/company DeleteCompany failed", zap.Error(err))
			response = responses.InternalErr
		}
		c.JSON(response.Code, &response)
		return
	}

	response = responses.Success
	c.JSON(response.Code, &response)
}

func (h *handler) GetCompanies(c *gin.Context) {
	var (
		filter   structs.CompanyFilter
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/company GetCompanies bind failed", zap.Error(err))
		response = responses.BadRequest
		return
	}

	companies, err := h.companySvs.GetCompanies(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/company GetCompanies failed", zap.Error(err))
		response = responses.InternalErr
		return
	}

	response = responses.Success
	response.Payload = companies
}
