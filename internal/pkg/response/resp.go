package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, Resp{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, "success", data)
}

func Error(c *gin.Context, code int, message string) {
	JSON(c, code, message, nil)
}
