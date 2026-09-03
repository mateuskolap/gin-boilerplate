package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	permissionUseCase domain.PermissionUseCase
}

func NewPermissionHandler(permissionUseCase domain.PermissionUseCase) *PermissionHandler {
	return &PermissionHandler{
		permissionUseCase: permissionUseCase,
	}
}

func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	params := extractPaginationParams(c)

	var filters []domain.Filter
	if name := c.Query("name"); name != "" {
		filters = append(filters, domain.Filter{
			Field:    "name",
			Operator: domain.OperatorILike,
			Value:    "%" + name + "%",
		})
	}

	result, err := h.permissionUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := dto.ToPaginatedResponse(result, dto.ToPermissionResponse)

	middleware.Success(c, 200, "Permissions retrieved successfully", response)
}
