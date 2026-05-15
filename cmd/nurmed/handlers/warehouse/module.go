package warehouse

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/fx"
	"go.uber.org/zap"

	restmiddleware "nurmed/cmd/nurmed/handlers/middleware"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
	warehousesvc "nurmed/internal/warehouse"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger       logger.ILogger
	WarehouseSvs warehousesvc.Service
}

type handler struct {
	logger       logger.ILogger
	warehouseSvs warehousesvc.Service
}

type Handler interface {
	GetWarehouses(c *gin.Context)
	GetStock(c *gin.Context)

	GetInventories(c *gin.Context)
	CreateInventory(c *gin.Context)
	PostInventory(c *gin.Context)

	GetTransfers(c *gin.Context)
	CreateTransfer(c *gin.Context)
	PostTransfer(c *gin.Context)

	GetWriteoffs(c *gin.Context)
	CreateWriteoff(c *gin.Context)
	PostWriteoff(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:       p.Logger,
		warehouseSvs: p.WarehouseSvs,
	}
}

func (h *handler) GetWarehouses(c *gin.Context) {
	scope := restmiddleware.GetRequestScope(c)
	warehouses, err := h.warehouseSvs.ListWarehouses(c.Request.Context(), scope.CompanyID)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/warehouse GetWarehouses", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = warehouses
	c.JSON(resp.Code, &resp)
}

func (h *handler) GetStock(c *gin.Context) {
	var filter structs.WarehouseStockFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse GetStock bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	stock, err := h.warehouseSvs.GetStock(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/warehouse GetStock", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = stock
	c.JSON(resp.Code, &resp)
}

// ---- Inventory ----

func (h *handler) GetInventories(c *gin.Context) {
	var filter structs.InventoryOrderFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse GetInventories bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	list, err := h.warehouseSvs.ListInventories(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/warehouse GetInventories", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = list
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreateInventory(c *gin.Context) {
	var req structs.CreateInventoryOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse CreateInventory bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if scope.CompanyID > 0 {
		if req.CompanyID == 0 {
			req.CompanyID = scope.CompanyID
		}
		if req.CompanyID != scope.CompanyID {
			resp := responses.Forbidden
			c.JSON(resp.Code, &resp)
			return
		}
	}
	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		req.CreatedBy = principal.UserID
	}
	order, err := h.warehouseSvs.CreateInventory(c.Request.Context(), req)
	if err != nil {
		h.handleWriteErr(c, "CreateInventory", err,
			warehousesvc.ErrInvalidInventoryPayload)
		return
	}
	resp := responses.Success
	resp.Payload = order
	c.JSON(resp.Code, &resp)
}

func (h *handler) PostInventory(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	var updatedBy int64
	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		updatedBy = principal.UserID
	}
	order, err := h.warehouseSvs.PostInventory(c.Request.Context(), id, scope.CompanyID, updatedBy)
	if err != nil {
		h.handlePostErr(c, "PostInventory", err)
		return
	}
	resp := responses.Success
	resp.Payload = order
	c.JSON(resp.Code, &resp)
}

// ---- Transfer ----

func (h *handler) GetTransfers(c *gin.Context) {
	var filter structs.TransferOrderFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse GetTransfers bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	list, err := h.warehouseSvs.ListTransfers(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/warehouse GetTransfers", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = list
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreateTransfer(c *gin.Context) {
	var req structs.CreateTransferOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse CreateTransfer bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if scope.CompanyID > 0 {
		if req.CompanyID == 0 {
			req.CompanyID = scope.CompanyID
		}
		if req.CompanyID != scope.CompanyID {
			resp := responses.Forbidden
			c.JSON(resp.Code, &resp)
			return
		}
	}
	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		req.CreatedBy = principal.UserID
	}
	order, err := h.warehouseSvs.CreateTransfer(c.Request.Context(), req)
	if err != nil {
		h.handleWriteErr(c, "CreateTransfer", err,
			warehousesvc.ErrInvalidTransferPayload)
		return
	}
	resp := responses.Success
	resp.Payload = order
	c.JSON(resp.Code, &resp)
}

func (h *handler) PostTransfer(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	order, err := h.warehouseSvs.PostTransfer(c.Request.Context(), id, scope.CompanyID)
	if err != nil {
		h.handlePostErr(c, "PostTransfer", err)
		return
	}
	resp := responses.Success
	resp.Payload = order
	c.JSON(resp.Code, &resp)
}

// ---- Writeoff ----

func (h *handler) GetWriteoffs(c *gin.Context) {
	var filter structs.WriteoffOrderFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse GetWriteoffs bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if filter.CompanyID == 0 && scope.CompanyID > 0 {
		filter.CompanyID = scope.CompanyID
	}
	list, err := h.warehouseSvs.ListWriteoffs(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/warehouse GetWriteoffs", zap.Error(err))
		resp := responses.InternalErr
		c.JSON(resp.Code, &resp)
		return
	}
	resp := responses.Success
	resp.Payload = list
	c.JSON(resp.Code, &resp)
}

func (h *handler) CreateWriteoff(c *gin.Context) {
	var req structs.CreateWriteoffOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/warehouse CreateWriteoff bind", zap.Error(err))
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if scope.CompanyID > 0 {
		if req.CompanyID == 0 {
			req.CompanyID = scope.CompanyID
		}
		if req.CompanyID != scope.CompanyID {
			resp := responses.Forbidden
			c.JSON(resp.Code, &resp)
			return
		}
	}
	if principal, ok := restmiddleware.GetPrincipal(c); ok {
		req.CreatedBy = principal.UserID
	}
	order, err := h.warehouseSvs.CreateWriteoff(c.Request.Context(), req)
	if err != nil {
		h.handleWriteErr(c, "CreateWriteoff", err,
			warehousesvc.ErrInvalidWriteoffPayload)
		return
	}
	resp := responses.Success
	resp.Payload = order
	c.JSON(resp.Code, &resp)
}

func (h *handler) PostWriteoff(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		resp := responses.BadRequest
		c.JSON(resp.Code, &resp)
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	order, err := h.warehouseSvs.PostWriteoff(c.Request.Context(), id, scope.CompanyID)
	if err != nil {
		h.handlePostErr(c, "PostWriteoff", err)
		return
	}
	resp := responses.Success
	resp.Payload = order
	c.JSON(resp.Code, &resp)
}

// ---- Shared helpers ----

func parseID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}

func (h *handler) handleWriteErr(c *gin.Context, op string, err error, validationErr error) {
	var pgErr *pgconn.PgError
	var resp structs.Response
	switch {
	case errors.Is(err, validationErr):
		resp = responses.BadRequest
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		resp = responses.Conflict
	case errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23514"):
		resp = responses.BadRequest
	case errors.Is(err, pgx.ErrNoRows):
		resp = responses.BadRequest
	default:
		h.logger.Error(c.Request.Context(), "handlers/warehouse "+op, zap.Error(err))
		resp = responses.InternalErr
	}
	c.JSON(resp.Code, &resp)
}

func (h *handler) handlePostErr(c *gin.Context, op string, err error) {
	var resp structs.Response
	switch {
	case errors.Is(err, warehousesvc.ErrOrderAlreadyPosted):
		resp = responses.Conflict
	case errors.Is(err, warehousesvc.ErrInsufficientStock):
		resp = responses.BadRequest
	case errors.Is(err, pgx.ErrNoRows):
		resp = responses.NotFound
	default:
		h.logger.Error(c.Request.Context(), "handlers/warehouse "+op, zap.Error(err))
		resp = responses.InternalErr
	}
	c.JSON(resp.Code, &resp)
}
