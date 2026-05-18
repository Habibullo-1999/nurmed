package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"

	intauth "nurmed/internal/auth"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
	"nurmed/pkg/logger"
)

var Module = fx.Provide(New)

type Params struct {
	fx.In
	Logger  logger.ILogger
	AuthSvs intauth.Service
}

type handler struct {
	logger  logger.ILogger
	authSvs intauth.Service
}

type Handler interface {
	Login(c *gin.Context)
	Refresh(c *gin.Context)
	Logout(c *gin.Context)
	CheckDevice(c *gin.Context)
	VerifyDevice(c *gin.Context)
}

func New(p Params) Handler {
	return &handler{
		logger:  p.Logger,
		authSvs: p.AuthSvs,
	}
}

func (h *handler) Login(c *gin.Context) {
	var (
		request  structs.LoginRequest
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/auth Login bind failed", zap.Error(err))
		response = responses.BadRequest
		return
	}

	tokens, err := h.authSvs.Login(c.Request.Context(), request, readAuthMeta(c))
	if err != nil {
		response = mapAuthErrorToResponse(err)
		if response.Code == responses.InternalErrCode {
			h.logger.Error(c.Request.Context(), "handlers/auth Login failed", zap.Error(err))
		}
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)
	if pendingToken := readCookieValue(c, h.authSvs.DevicePendingCookieName()); pendingToken != "" {
		trustedToken, trustErr := h.authSvs.TrustPendingDevice(c.Request.Context(), pendingToken)
		if trustErr != nil {
			h.logger.Warning(c.Request.Context(), "handlers/auth trust pending device failed", zap.Error(trustErr))
		} else if trustedToken != "" {
			h.setDeviceTrustCookie(c, trustedToken)
			h.clearDevicePendingCookie(c)
		}
	}

	response = responses.Success
	response.Payload = structs.AuthResponse{
		AccessToken:          tokens.AccessToken,
		AccessTokenExpiresAt: tokens.AccessTokenExpiresAt,
		TokenType:            "Bearer",
		UserID:               tokens.UserID,
		UserName:             tokens.UserName,
		CompanyID:            tokens.CompanyID,
	}
}

func (h *handler) Refresh(c *gin.Context) {
	var (
		request  structs.RefreshRequest
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	refreshToken := readRefreshToken(c, h.authSvs.RefreshCookieName())
	if refreshToken == "" && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err == nil {
			refreshToken = request.RefreshToken
		}
	}

	tokens, err := h.authSvs.Refresh(c.Request.Context(), refreshToken, readAuthMeta(c))
	if err != nil {
		response = mapAuthErrorToResponse(err)
		if response.Code == responses.InternalErrCode {
			h.logger.Error(c.Request.Context(), "handlers/auth Refresh failed", zap.Error(err))
		}
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)

	response = responses.Success
	response.Payload = structs.AuthResponse{
		AccessToken:          tokens.AccessToken,
		AccessTokenExpiresAt: tokens.AccessTokenExpiresAt,
		TokenType:            "Bearer",
		UserID:               tokens.UserID,
		UserName:             tokens.UserName,
		CompanyID:            tokens.CompanyID,
	}
}

func (h *handler) Logout(c *gin.Context) {
	var (
		request  structs.LogoutRequest
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	refreshToken := readRefreshToken(c, h.authSvs.RefreshCookieName())
	if refreshToken == "" && c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err == nil {
			refreshToken = request.RefreshToken
		}
	}

	var userID *int64
	if rawUserID, exists := c.Get("auth.user_id"); exists {
		if value, ok := rawUserID.(int64); ok {
			userID = &value
		}
	}

	if err := h.authSvs.Logout(c.Request.Context(), refreshToken, userID); err != nil {
		h.logger.Error(c.Request.Context(), "handlers/auth Logout failed", zap.Error(err))
		response = responses.InternalErr
		return
	}

	h.clearRefreshCookie(c)
	response = responses.Success
	response.Payload = gin.H{"loggedOut": true}
}

func (h *handler) setRefreshCookie(c *gin.Context, token string) {
	h.setCookie(c, h.authSvs.RefreshCookieName(), token, int(h.authSvs.RefreshTokenTTL().Seconds()))
}

func (h *handler) clearRefreshCookie(c *gin.Context) {
	h.clearCookie(c, h.authSvs.RefreshCookieName())
}

func (h *handler) setDeviceTrustCookie(c *gin.Context, token string) {
	h.setCookie(c, h.authSvs.DeviceTrustCookieName(), token, int(h.authSvs.DeviceTrustCookieTTL().Seconds()))
}

func (h *handler) setDevicePendingCookie(c *gin.Context, token string) {
	h.setCookie(c, h.authSvs.DevicePendingCookieName(), token, int(h.authSvs.DevicePendingCookieTTL().Seconds()))
}

func (h *handler) clearDevicePendingCookie(c *gin.Context) {
	h.clearCookie(c, h.authSvs.DevicePendingCookieName())
}

func (h *handler) setCookie(c *gin.Context, name, value string, maxAge int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		value,
		maxAge,
		"/",
		h.authSvs.RefreshCookieDomain(),
		h.authSvs.RefreshCookieSecure(),
		true,
	)
}

func (h *handler) clearCookie(c *gin.Context, name string) {
	h.setCookie(c, name, "", -1)
}

func readAuthMeta(c *gin.Context) structs.AuthMeta {
	return structs.AuthMeta{
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

func readRefreshToken(c *gin.Context, cookieName string) string {
	if token, err := c.Cookie(cookieName); err == nil && token != "" {
		return token
	}
	return c.GetHeader("X-Refresh-Token")
}

func readCookieValue(c *gin.Context, cookieName string) string {
	value, _ := c.Cookie(cookieName)
	return value
}

func mapAuthErrorToResponse(err error) structs.Response {
	switch {
	case errors.Is(err, intauth.ErrInvalidCredentials):
		return responses.Unauthorized
	case errors.Is(err, intauth.ErrUnauthorized):
		return responses.Unauthorized
	case errors.Is(err, intauth.ErrForbidden):
		return responses.Forbidden
	case errors.Is(err, intauth.ErrUserLocked):
		return responses.Forbidden
	default:
		return responses.InternalErr
	}
}
