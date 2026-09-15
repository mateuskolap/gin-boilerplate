package response

import "github.com/gin-gonic/gin"

type ApiResponse struct {
	Success bool              `json:"success" example:"true"`
	Message string            `json:"message,omitempty" example:"Operation completed successfully"`
	Data    interface{}       `json:"data,omitempty"`
	Error   string            `json:"error,omitempty" example:"Error description"`
	Errors  map[string]string `json:"errors,omitempty"`
}

func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}
