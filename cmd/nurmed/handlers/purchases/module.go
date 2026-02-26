package purchases

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/fx"
	"go.uber.org/zap"

	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	"nurmed/internal/purchases"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger      logger.ILogger
	PurchaseSvs purchases.Service
}

type handler struct {
	logger      logger.ILogger
	purchaseSvs purchases.Service
}

type Handler interface {
	GetAcquisitions(c *gin.Context)
	CreateAcquisition(c *gin.Context)
	GetRegistry(c *gin.Context)
	GetReturns(c *gin.Context)
	CreateReturn(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:      p.Logger,
		purchaseSvs: p.PurchaseSvs,
	}
}

func (h *handler) GetAcquisitions(c *gin.Context) {
	h.listOrders(c, h.purchaseSvs.ListAcquisitions)
}

func (h *handler) CreateAcquisition(c *gin.Context) {
	h.createOrder(c, h.purchaseSvs.CreateAcquisition)
}

func (h *handler) GetRegistry(c *gin.Context) {
	h.listOrders(c, h.purchaseSvs.ListRegistry)
}

func (h *handler) GetReturns(c *gin.Context) {
	var request structs.PurchaseReturnFilter
	if err := c.ShouldBindQuery(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/purchases/module.go err from c.ShouldBindQuery()", zap.Error(err))
		response := responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	if request.CompanyID == 0 && scope.CompanyID > 0 {
		request.CompanyID = scope.CompanyID
	}

	purchaseReturns, err := h.purchaseSvs.ListReturns(c.Request.Context(), request)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/purchases/module.go err from h.purchaseSvs.ListReturns()", zap.Error(err))
		response := responses.InternalErr
		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = purchaseReturns
	c.JSON(response.Code, &response)
}

func (h *handler) CreateReturn(c *gin.Context) {
	var request structs.PurchaseCreateReturnRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/purchases/module.go err from c.ShouldBindJSON()", zap.Error(err))
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

	purchaseReturn, err := h.purchaseSvs.CreateReturn(c.Request.Context(), request)
	if err != nil {
		var pgErr *pgconn.PgError
		var response structs.Response

		switch {
		case errors.Is(err, purchases.ErrInvalidPurchaseReturnPayload):
			response = responses.BadRequest
		case errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514"):
			response = responses.BadRequest
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/purchases/module.go err from h.purchaseSvs.CreateReturn()", zap.Error(err))
			response = responses.InternalErr
		}

		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = purchaseReturn
	c.JSON(response.Code, &response)
}

func (h *handler) listOrders(c *gin.Context, listFn func(ctx context.Context, request structs.PurchaseOrderFilter) ([]structs.PurchaseOrderResponse, error)) {
	var request structs.PurchaseOrderFilter
	if err := c.ShouldBindQuery(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/purchases/module.go err from c.ShouldBindQuery()", zap.Error(err))
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
		h.logger.Error(c.Request.Context(), "handlers/purchases/module.go err from listFn()", zap.Error(err))
		response := responses.InternalErr
		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = orders
	c.JSON(response.Code, &response)
}

func (h *handler) createOrder(c *gin.Context, createFn func(ctx context.Context, request structs.PurchaseCreateOrderRequest) (structs.PurchaseOrderResponse, error)) {
	var request structs.PurchaseCreateOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/purchases/module.go err from c.ShouldBindJSON()", zap.Error(err))
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
		case errors.Is(err, purchases.ErrInvalidPurchasePayload):
			response = responses.BadRequest
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			response = responses.Conflict
		case errors.As(err, &pgErr) && pgErr.Code == "23514":
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/purchases/module.go err from createFn()", zap.Error(err))
			response = responses.InternalErr
		}

		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = order
	c.JSON(response.Code, &response)
}
