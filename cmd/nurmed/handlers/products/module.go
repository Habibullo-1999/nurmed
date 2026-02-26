package products

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/fx"
	"go.uber.org/zap"

	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	"nurmed/internal/products"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger     logger.ILogger
	ProductSvs products.Service
}

type handler struct {
	logger     logger.ILogger
	productSvs products.Service
}

type Handler interface {
	GetProducts(c *gin.Context)
	CreateProduct(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:     p.Logger,
		productSvs: p.ProductSvs,
	}
}

func (h *handler) GetProducts(c *gin.Context) {
	var request structs.ProductFilter
	if err := c.ShouldBindQuery(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/products/module.go err from c.ShouldBindQuery()", zap.Error(err))
		response := responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	if request.CompanyID == 0 && scope.CompanyID > 0 {
		request.CompanyID = scope.CompanyID
	}

	productsList, err := h.productSvs.ListProducts(c.Request.Context(), request)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/products/module.go err from h.productSvs.ListProducts()", zap.Error(err))
		response := responses.InternalErr
		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = productsList
	c.JSON(response.Code, &response)
}

func (h *handler) CreateProduct(c *gin.Context) {
	var request structs.CreateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/products/module.go err from c.ShouldBindJSON()", zap.Error(err))
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

	product, err := h.productSvs.CreateProduct(c.Request.Context(), request)
	if err != nil {
		var pgErr *pgconn.PgError
		var response structs.Response

		switch {
		case errors.Is(err, products.ErrInvalidProductPayload):
			response = responses.BadRequest
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			response = responses.Conflict
		case errors.As(err, &pgErr) && pgErr.Code == "23514":
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/products/module.go err from h.productSvs.CreateProduct()", zap.Error(err))
			response = responses.InternalErr
		}

		c.JSON(response.Code, &response)
		return
	}

	response := responses.Success
	response.Payload = product
	c.JSON(response.Code, &response)
}
