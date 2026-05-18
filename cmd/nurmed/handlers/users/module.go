package users

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
	"nurmed/internal/users"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger  logger.ILogger
	UserSvs users.Service
}

type handler struct {
	logger  logger.ILogger
	userSvs users.Service
}

type Handler interface {
	GetUsers(c *gin.Context)
	CreateUser(c *gin.Context)
	AdminCreateUser(c *gin.Context)
	UpdateUser(c *gin.Context)
	DeleteUser(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:  p.Logger,
		userSvs: p.UserSvs,
	}
}

func (h *handler) GetUsers(c *gin.Context) {
	var (
		request  structs.UserFilter
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	err := c.ShouldBindQuery(&request)
	if err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/users/module.go err from c.ShouldBindQuery()", zap.Error(err))
		response = responses.BadRequest
		return
	}
	scope := restmiddleware.GetRequestScope(c)
	if request.CompanyID == 0 && scope.CompanyID > 0 {
		request.CompanyID = scope.CompanyID
	}

	usersList, err := h.userSvs.GetUsers(c.Request.Context(), request)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/users/module.go err from h.userSvs.GetUsers()", zap.Error(err))
		response = responses.InternalErr
		return
	}

	response = responses.Success
	response.Payload = usersList
}

// CreateUser создаёт пользователя в рамках scope авторизованного пользователя.
// companyId берётся из scope (JWT), из тела не принимается.
func (h *handler) CreateUser(c *gin.Context) {
	var (
		request  structs.CreateUserRequest
		response structs.Response
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/users/module.go err from c.ShouldBindJSON()", zap.Error(err))
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	scope := restmiddleware.GetRequestScope(c)
	request.CompanyID = scope.CompanyID

	if request.Role.ScopeType == "company" && request.Role.ScopeID != nil && scope.CompanyID > 0 && *request.Role.ScopeID != scope.CompanyID {
		response = responses.Forbidden
		c.JSON(response.Code, &response)
		return
	}

	h.doCreateUser(c, request)
}

// AdminCreateUser создаёт пользователя с явным companyId из тела запроса.
// Доступно только суперадмину через /admin/users.
func (h *handler) AdminCreateUser(c *gin.Context) {
	var (
		request  structs.CreateUserRequest
		response structs.Response
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/users/module.go err from c.ShouldBindJSON()", zap.Error(err))
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	h.doCreateUser(c, request)
}

func (h *handler) doCreateUser(c *gin.Context, request structs.CreateUserRequest) {
	var response structs.Response

	user, err := h.userSvs.CreateUser(c.Request.Context(), request)
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, users.ErrInvalidUserPayload):
			response = responses.BadRequest
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			response = responses.Conflict
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.BadRequest
		default:
			h.logger.Error(c.Request.Context(), "handlers/users/module.go err from h.userSvs.CreateUser()", zap.Error(err))
			response = responses.InternalErr
		}
		c.JSON(response.Code, &response)
		return
	}

	response = responses.Success
	response.Payload = user
	c.JSON(response.Code, &response)
}

func (h *handler) UpdateUser(c *gin.Context) {
	var (
		request  structs.UpdateUserRequest
		response structs.Response
	)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/users UpdateUser bind failed", zap.Error(err))
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	updated, err := h.userSvs.UpdateUser(c.Request.Context(), id, request)
	if err != nil {
		switch {
		case errors.Is(err, users.ErrInvalidUserPayload):
			response = responses.BadRequest
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.NotFound
		default:
			h.logger.Error(c.Request.Context(), "handlers/users UpdateUser failed", zap.Error(err))
			response = responses.InternalErr
		}
		c.JSON(response.Code, &response)
		return
	}

	response = responses.Success
	response.Payload = updated
	c.JSON(response.Code, &response)
}

func (h *handler) DeleteUser(c *gin.Context) {
	var response structs.Response

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response = responses.BadRequest
		c.JSON(response.Code, &response)
		return
	}

	if err := h.userSvs.DeleteUser(c.Request.Context(), id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			response = responses.NotFound
		default:
			h.logger.Error(c.Request.Context(), "handlers/users DeleteUser failed", zap.Error(err))
			response = responses.InternalErr
		}
		c.JSON(response.Code, &response)
		return
	}

	response = responses.Success
	c.JSON(response.Code, &response)
}
