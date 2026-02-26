package sales

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/fx"
	"go.uber.org/zap"

	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	"nurmed/internal/responses"
	"nurmed/internal/sales"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger   logger.ILogger
	SalesSvs sales.Service
}

type handler struct {
	logger   logger.ILogger
	salesSvs sales.Service
}

type Handler interface {
	GetRealizations(c *gin.Context)
	CreateRealization(c *gin.Context)
	GetRegistry(c *gin.Context)
	GetMobileSales(c *gin.Context)
	CreateMobileSale(c *gin.Context)
	GetPOSSales(c *gin.Context)
	CreatePOSSale(c *gin.Context)
	GetReturns(c *gin.Context)
	CreateReturn(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:   p.Logger,
		salesSvs: p.SalesSvs,
	}
}

func (h *handler) GetRealizations(c *gin.Context) {
	h.listOrders(c, h.salesSvs.ListRealizations)
}

func (h *handler) CreateRealization(c *gin.Context) {
	h.createOrder(c, h.salesSvs.CreateRealization)
}

func (h *handler) GetRegistry(c *gin.Context) {
	h.listOrders(c, h.salesSvs.ListRegistry)
}

func (h *handler) GetMobileSales(c *gin.Context) {
	h.listOrders(c, h.salesSvs.ListMobile)
}

func (h *handler) CreateMobileSale(c *gin.Context) {
	h.createOrder(c, h.salesSvs.CreateMobile)
}

func (h *handler) GetPOSSales(c *gin.Context) {
	h.listOrders(c, h.salesSvs.ListPOS)
}

func (h *handler) CreatePOSSale(c *gin.Context) {
	h.createOrder(c, h.salesSvs.CreatePOS)
}

func (h *handler) GetReturns(c *gin.Context) {
	var request structs.SalesReturnFilter
	if err := c.ShouldBindQuery(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/sales/module.go err from c.ShouldBindQuery()", zap.Error(err))
		response := responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	if request.CompanyID == 0 && scope.CompanyID > 0 {
		request.CompanyID = scope.CompanyID
	}

	salesReturns, err := h.salesSvs.ListReturns(c.Request.Context(), request)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/sales/module.go err from h.salesSvs.ListReturns()", zap.Error(err))
		response := responses.InternalErr
		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = salesReturns
	c.JSON(response.Code, &response)
}

func (h *handler) CreateReturn(c *gin.Context) {
	var request structs.SalesCreateReturnRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/sales/module.go err from c.ShouldBindJSON()", zap.Error(err))
		response := responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	if scope.CompanyID > 0 {
		if request.CompanyID == 0 {
			request.CompanyID = scope.CompanyID
		}
		if request.CompanyID != scope.CompanyID {
			response := responses.Forbidden
			c.JSON(response.Code, &response)
			return
		}
	}

	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		request.CreatedBy = principal.UserID
	}

	salesReturn, err := h.salesSvs.CreateReturn(c.Request.Context(), request)
	if err != nil {
		var pgErr *pgconn.PgError
		var response structs.Response

		switch {
		case errors.Is(err, sales.ErrInvalidReturnPayload):
			response = responses.BadRequest
		case errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514"):
			response = responses.BadRequest
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/sales/module.go err from h.salesSvs.CreateReturn()", zap.Error(err))
			response = responses.InternalErr
		}

		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = salesReturn
	c.JSON(response.Code, &response)
}

func (h *handler) listOrders(c *gin.Context, listFn func(ctx context.Context, request structs.SalesOrderFilter) ([]structs.SalesOrderResponse, error)) {
	var request structs.SalesOrderFilter
	if err := c.ShouldBindQuery(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/sales/module.go err from c.ShouldBindQuery()", zap.Error(err))
		response := responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	if request.CompanyID == 0 && scope.CompanyID > 0 {
		request.CompanyID = scope.CompanyID
	}

	orders, err := listFn(c.Request.Context(), request)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/sales/module.go err from listFn()", zap.Error(err))
		response := responses.InternalErr
		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = orders
	c.JSON(response.Code, &response)
}

func (h *handler) createOrder(c *gin.Context, createFn func(ctx context.Context, request structs.SalesCreateOrderRequest) (structs.SalesOrderResponse, error)) {
	var request structs.SalesCreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/sales/module.go err from c.ShouldBindJSON()", zap.Error(err))
		response := responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	if scope.CompanyID > 0 {
		if request.CompanyID == 0 {
			request.CompanyID = scope.CompanyID
		}
		if request.CompanyID != scope.CompanyID {
			response := responses.Forbidden
			c.JSON(response.Code, &response)
			return
		}
	}

	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		request.CreatedBy = principal.UserID
	}

	order, err := createFn(c.Request.Context(), request)
	if err != nil {
		var pgErr *pgconn.PgError
		var response structs.Response
		switch {
		case errors.Is(err, sales.ErrInvalidSalesPayload):
			response = responses.BadRequest
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			response = responses.Conflict
		case errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514"):
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/sales/module.go err from createFn()", zap.Error(err))
			response = responses.InternalErr
		}

		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = order
	c.JSON(response.Code, &response)
}
