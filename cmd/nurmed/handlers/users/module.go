package users

import (
	"errors"

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
	if scope.CompanyID > 0 {
		if request.CompanyID == 0 {
			request.CompanyID = scope.CompanyID
		}
		if request.CompanyID != scope.CompanyID {
			response = responses.Forbidden
			c.JSON(response.Code, &response)
			return
		}
		if request.Role.ScopeType == "company" && request.Role.ScopeID != nil && *request.Role.ScopeID != scope.CompanyID {
			response = responses.Forbidden
			c.JSON(response.Code, &response)
			return
		}
	}

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
