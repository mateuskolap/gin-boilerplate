package v1

import (
	"context"
	"net/http"
	"time"

	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain/port"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	checker port.HealthChecker
}

func NewHealthHandler(checker port.HealthChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	response.Success(c, http.StatusOK, "Service is alive", nil)
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.checker.Readiness(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, response.ApiResponse{
			Success: false,
			Error:   "Service is not ready",
		})
		return
	}

	response.Success(c, http.StatusOK, "Service is ready", nil)
}
