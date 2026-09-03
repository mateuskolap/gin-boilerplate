package middleware

import (
	"errors"
	"gin-boilerplate/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message,omitempty" example:"Operation completed successfully"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty" example:"Error description"`
}

func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 && !c.Writer.Written() {
			err := c.Errors.Last().Err
			HandleError(c, err)
		}
	}
}

func HandleError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		statusCode := http.StatusInternalServerError

		switch appErr.Type {
		case domain.ErrTypeNotFound:
			statusCode = http.StatusNotFound
		case domain.ErrTypeConflict:
			statusCode = http.StatusConflict
		case domain.ErrTypeUnauthorized:
			statusCode = http.StatusUnauthorized
		case domain.ErrTypeForbidden:
			statusCode = http.StatusForbidden
		case domain.ErrTypeValidation:
			statusCode = http.StatusUnprocessableEntity
		case domain.ErrTypeInternal:
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, ApiResponse{
			Success: false,
			Error:   appErr.Message,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, ApiResponse{
		Success: false,
		Error:   "Internal server error",
	})
}
