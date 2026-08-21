package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RoleHandler struct {
	roleUseCase domain.RoleUseCase
}

func NewRoleHandler(roleUseCase domain.RoleUseCase) *RoleHandler {
	return &RoleHandler{
		roleUseCase: roleUseCase,
	}
}

func (h *RoleHandler) FindRole(c *gin.Context) {
	roleID, _ := uuid.Parse(c.Param("id"))

	role, err := h.roleUseCase.Find(c.Request.Context(), roleID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Role retrieved successfully", dto.ToRoleResponse(role))
}

// ListRoles godoc
// @Summary      List roles
// @Description  Get paginated list of roles with optional filters and sorting
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int     false  "Page number (default: 1)"
// @Param        limit  query     int     false  "Items per page (default: 10, max: 100)"
// @Param        sort   query     string  false  "Sorting criteria (e.g. name:asc, created_at:desc or -created_at)"
// @Param        name   query     string  false  "Filter by role name (partial match)"
// @Success      200    {object}  middleware.ApiResponse{data=dto.PaginatedRoleResponse}
// @Failure      401    {object}  middleware.ApiResponse
// @Failure      422    {object}  middleware.ApiResponse
// @Failure      500    {object}  middleware.ApiResponse
// @Router       /api/v1/roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	params := extractPaginationParams(c)

	var filters []domain.Filter
	if name := c.Query("name"); name != "" {
		filters = append(filters, domain.Filter{
			Field:    "name",
			Operator: domain.OperatorILike,
			Value:    "%" + name + "%",
		})
	}

	result, err := h.roleUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := dto.ToPaginatedResponse(result, dto.ToRoleResponse)
	middleware.Success(c, http.StatusOK, "Roles retrieved successfully", response)
}
