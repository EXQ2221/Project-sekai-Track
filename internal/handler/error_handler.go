package handler

import (
	"errors"

	"Project_sekai_search/internal/pkg/errcode"
	"Project_sekai_search/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errcode.ErrUsernameIncorrect):
		response.Error(c, 401, errcode.ErrUsernameIncorrect.Error())
	case errors.Is(err, errcode.ErrPasswordIncorrect):
		response.Error(c, 401, errcode.ErrPasswordIncorrect.Error())
	case errors.Is(err, errcode.ErrSessionRevoked):
		response.Error(c, 401, errcode.ErrSessionRevoked.Error())
	case errors.Is(err, errcode.ErrRefreshReuse):
		response.Error(c, 403, errcode.ErrRefreshReuse.Error())
	case errors.Is(err, errcode.ErrDeviceMismatch):
		response.Error(c, 401, errcode.ErrDeviceMismatch.Error())
	case errors.Is(err, errcode.ErrBadRequest):
		response.Error(c, 400, errcode.ErrBadRequest.Error())
	case errors.Is(err, errcode.ErrUnauthorized):
		response.Error(c, 401, errcode.ErrUnauthorized.Error())
	case errors.Is(err, errcode.ErrForbidden):
		response.Error(c, 403, errcode.ErrForbidden.Error())
	case errors.Is(err, errcode.ErrConflict):
		response.Error(c, 409, errcode.ErrConflict.Error())
	case errors.Is(err, errcode.ErrNotFound):
		response.Error(c, 404, errcode.ErrNotFound.Error())
	case errors.Is(err, errcode.ErrInternal):
		response.Error(c, 500, errcode.ErrInternal.Error())
	default:
		response.Error(c, 500, err.Error())
	}
}
