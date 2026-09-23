package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"net/http"

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

// ListPermissions godoc
// @Summary      List permissions
// @Description  Get paginated list of permissions with optional filtering and sorting. Requires 'view_permission' permission.
// @Tags         Permissions
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int     false  "Page number (default: 1)" minimum(1)
// @Param        limit  query     int     false  "Items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param        sort   query     string  false  "Sorting criteria (e.g. name:asc, created_at:desc or -created_at)"
// @Param        name   query     string  false  "Filter by permission name (partial match)"
// @Success      200    {object}  response.ApiResponse{data=dto.PaginatedPermissionResponse} "Permissions retrieved successfully"
// @Failure      401    {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403    {object}  response.ApiResponse "Forbidden - Requires view_permission permission"
// @Failure      422    {object}  response.ApiResponse "Unprocessable Entity - Invalid filter or sorting parameter"
// @Failure      500    {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/permissions [get]
func (h *PermissionHandler) ListPermissions(c *gin.Context) {
	params, err := extractPaginationParams(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var filters []shared.Filter
	if name := c.Query("name"); name != "" {
		filters = append(filters, shared.Filter{
			Field:    "name",
			Operator: shared.OperatorILike,
			Value:    "%" + name + "%",
		})
	}

	result, err := h.permissionUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := dto.ToPaginatedResponse(result, dto.ToPermissionResponse)

	response.Success(c, http.StatusOK, "Permissions retrieved successfully", resp)
}
