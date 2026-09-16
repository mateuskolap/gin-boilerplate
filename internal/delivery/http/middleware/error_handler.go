package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var statusMap = map[domain.ErrorType]int{
	domain.ErrTypeNotFound:        http.StatusNotFound,
	domain.ErrTypeConflict:        http.StatusConflict,
	domain.ErrTypeUnauthorized:    http.StatusUnauthorized,
	domain.ErrTypeForbidden:       http.StatusForbidden,
	domain.ErrTypeValidation:      http.StatusUnprocessableEntity,
	domain.ErrTypeTooManyRequests: http.StatusTooManyRequests,
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		var appErr *domain.AppError
		if !errors.As(c.Errors.Last().Err, &appErr) {
			c.JSON(http.StatusInternalServerError, response.ApiResponse{
				Success: false,
				Error:   "Internal server error",
			})
			return
		}

		status := statusMap[appErr.Type]
		if status == 0 {
			status = http.StatusInternalServerError
		}

		res := response.ApiResponse{
			Success: false,
			Error:   appErr.Message,
		}

		var ve validator.ValidationErrors
		if errors.As(appErr.Err, &ve) && Translator != nil {
			res.Errors = ve.Translate(Translator)
		}

		var ute *json.UnmarshalTypeError
		if errors.As(appErr.Err, &ute) {
			field := ute.Field
			if field == "" {
				field = "payload"
			}
			res.Errors = map[string]string{
				field: fmt.Sprintf("%s must be of type %s", field, ute.Type.String()),
			}
		}

		c.JSON(status, res)
	}
}
