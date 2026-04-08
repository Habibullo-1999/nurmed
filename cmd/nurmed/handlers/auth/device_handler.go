package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	intauth "nurmed/internal/auth"
	"nurmed/internal/responses"
	"nurmed/internal/structs"
)

func (h *handler) CheckDevice(c *gin.Context) {
	var (
		request  structs.DeviceCheckRequest
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/auth CheckDevice bind failed", zap.Error(err))
		response = responses.BadRequest
		return
	}

	checkResponse, pendingToken, err := h.authSvs.CheckDevice(
		c.Request.Context(),
		request.DeviceInfo,
		readCookieValue(c, h.authSvs.DeviceTrustCookieName()),
	)
	if err != nil {
		h.logger.Error(c.Request.Context(), "handlers/auth CheckDevice failed", zap.Error(err))
		response = responses.InternalErr
		return
	}

	if pendingToken != "" {
		h.setDevicePendingCookie(c, pendingToken)
	}

	response = responses.Success
	response.Payload = checkResponse
}

func (h *handler) VerifyDevice(c *gin.Context) {
	var (
		request  structs.DeviceVerificationRequest
		response structs.Response
	)

	defer c.JSON(response.Code, &response)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warning(c.Request.Context(), "handlers/auth VerifyDevice bind failed", zap.Error(err))
		response = responses.BadRequest
		return
	}

	trustedToken, codeSent, err := h.authSvs.VerifyDevice(
		c.Request.Context(),
		request,
		readCookieValue(c, h.authSvs.DevicePendingCookieName()),
	)
	if err != nil {
		switch {
		case errors.Is(err, intauth.ErrPendingDeviceNotFound):
			response = responses.Unauthorized
		case errors.Is(err, intauth.ErrVerificationCodeInvalid):
			response = responses.Unauthorized
		case errors.Is(err, intauth.ErrVerificationCodeExpired):
			response = responses.NewResponse(410, "verification code expired")
		case errors.Is(err, intauth.ErrForbidden):
			response = responses.Forbidden
		default:
			h.logger.Error(c.Request.Context(), "handlers/auth VerifyDevice failed", zap.Error(err))
			response = responses.InternalErr
		}
		return
	}

	if trustedToken != "" {
		h.setDeviceTrustCookie(c, trustedToken)
		h.clearDevicePendingCookie(c)
		response = responses.Success
		response.Payload = structs.DeviceVerificationResponse{
			Verified: true,
			Message:  "device trusted",
		}
		return
	}

	response = responses.Success
	response.Payload = structs.DeviceVerificationResponse{
		Verified: false,
		CodeSent: codeSent,
		Message:  "verification code sent",
	}
}
