package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain/shared"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var statusMap = map[shared.ErrorType]int{
	shared.ErrTypeNotFound:        http.StatusNotFound,
	shared.ErrTypeConflict:        http.StatusConflict,
	shared.ErrTypeUnauthorized:    http.StatusUnauthorized,
	shared.ErrTypeForbidden:       http.StatusForbidden,
	shared.ErrTypeValidation:      http.StatusUnprocessableEntity,
	shared.ErrTypeTooManyRequests: http.StatusTooManyRequests,
	shared.ErrTypeUnavailable:     http.StatusServiceUnavailable,
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		var appErr *shared.AppError
		if !errors.As(c.Errors.Last().Err, &appErr) {
			slog.ErrorContext(c.Request.Context(), "unhandled request error",
				"request_id", requestIDFromContext(c),
				"error", c.Errors.Last().Err,
			)
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
		if status >= http.StatusInternalServerError {
			slog.ErrorContext(c.Request.Context(), "request failed",
				"request_id", requestIDFromContext(c),
				"error_type", appErr.Type,
				"error", appErr.Err,
			)
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
