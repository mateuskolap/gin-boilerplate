package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	roleUseCase domain.RoleUseCase
}

func NewRoleHandler(roleUseCase domain.RoleUseCase) *RoleHandler {
	return &RoleHandler{
		roleUseCase: roleUseCase,
	}
}

// FindRole godoc
// @Summary      Get role by ID
// @Description  Retrieve detailed information of a role by UUID
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Role UUID" format(uuid)
// @Success      200  {object}  middleware.ApiResponse{data=dto.RoleResponse}
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      404  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/roles/{id} [get]
func (h *RoleHandler) FindRole(c *gin.Context) {
	roleID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

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

// CreateRole godoc
// @Summary      Create a new role
// @Description  Create a new role with the specified name
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.CreateRoleRequest true "Role creation details"
// @Success      201  {object}  middleware.ApiResponse{data=dto.RoleResponse}
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      409  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	req, err := bindJSON[dto.CreateRoleRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	role := &domain.Role{
		Name: req.Name,
	}

	if err := h.roleUseCase.Create(c.Request.Context(), role); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusCreated, "Role created successfully", dto.ToRoleResponse(role))
}

// UpdateRole godoc
// @Summary      Update role
// @Description  Update role information by UUID
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                 true  "Role UUID" format(uuid)
// @Param        request  body      dto.UpdateRoleRequest  true  "Role update details"
// @Success      200      {object}  middleware.ApiResponse{data=dto.RoleResponse}
// @Failure      401      {object}  middleware.ApiResponse
// @Failure      404      {object}  middleware.ApiResponse
// @Failure      422      {object}  middleware.ApiResponse
// @Failure      500      {object}  middleware.ApiResponse
// @Router       /api/v1/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	roleID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdateRoleRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	role := &domain.Role{
		ID:   roleID,
		Name: req.Name,
	}

	if err := h.roleUseCase.Update(c.Request.Context(), role); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Role updated successfully", dto.ToRoleResponse(role))
}

// DeleteRole godoc
// @Summary      Delete role
// @Description  Delete a role by UUID
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Role UUID" format(uuid)
// @Success      200  {object}  middleware.ApiResponse
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      404  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	roleID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.roleUseCase.Delete(c.Request.Context(), roleID); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Role deleted successfully", nil)
}

// AddPermissions godoc
// @Summary      Add permissions to role
// @Description  Assign one or more permissions to a role by permission UUIDs
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                        true  "Role UUID" format(uuid)
// @Param        request  body      dto.UpdatePermissionsRequest  true  "Permission UUIDs to assign"
// @Success      201      {object}  middleware.ApiResponse
// @Failure      401      {object}  middleware.ApiResponse
// @Failure      404      {object}  middleware.ApiResponse
// @Failure      422      {object}  middleware.ApiResponse
// @Failure      500      {object}  middleware.ApiResponse
// @Router       /api/v1/roles/{id}/permissions [post]
func (h *RoleHandler) AddPermissions(c *gin.Context) {
	roleID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdatePermissionsRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.roleUseCase.AddPermissions(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusCreated, "Permissions added successfully", nil)
}

// RemovePermissions godoc
// @Summary      Remove permissions from role
// @Description  Remove one or more permissions from a role by permission UUIDs
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                        true  "Role UUID" format(uuid)
// @Param        request  body      dto.UpdatePermissionsRequest  true  "Permission UUIDs to remove"
// @Success      200      {object}  middleware.ApiResponse
// @Failure      401      {object}  middleware.ApiResponse
// @Failure      404      {object}  middleware.ApiResponse
// @Failure      422      {object}  middleware.ApiResponse
// @Failure      500      {object}  middleware.ApiResponse
// @Router       /api/v1/roles/{id}/permissions [delete]
func (h *RoleHandler) RemovePermissions(c *gin.Context) {
	roleID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdatePermissionsRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.roleUseCase.RemovePermissions(c.Request.Context(), roleID, req.PermissionIDs); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Permissions removed successfully", nil)
}
