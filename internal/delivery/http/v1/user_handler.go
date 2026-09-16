package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userUseCase domain.UserUseCase
}

func NewUserHandler(userUseCase domain.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

// FindUser godoc
// @Summary      Get user by ID
// @Description  Retrieve detailed user profile including assigned roles by UUID. Requires 'view_user' permission.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User UUID" format(uuid)
// @Success      200  {object}  response.ApiResponse{data=dto.UserWithRoleResponse} "User retrieved successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403  {object}  response.ApiResponse "Forbidden - Requires view_user permission"
// @Failure      404  {object}  response.ApiResponse "Not Found - User not found"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) FindUser(c *gin.Context) {
	userID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	user, err := h.userUseCase.Find(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "User retrieved successfully", dto.ToUserWithRoleResponse(user))
}

// ListUsers godoc
// @Summary      List users
// @Description  Get paginated list of users with optional filtering and sorting. Requires 'view_user' permission.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int     false  "Page number (default: 1)" minimum(1)
// @Param        limit  query     int     false  "Items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param        sort   query     string  false  "Sorting criteria (e.g. name:asc, created_at:desc or -created_at)"
// @Param        name   query     string  false  "Filter by user name (partial match)"
// @Param        email  query     string  false  "Filter by user email (partial match)"
// @Success      200    {object}  response.ApiResponse{data=dto.PaginatedUserResponse} "Users retrieved successfully"
// @Failure      401    {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403    {object}  response.ApiResponse "Forbidden - Requires view_user permission"
// @Failure      422    {object}  response.ApiResponse "Unprocessable Entity - Invalid filter or sorting parameter"
// @Failure      500    {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	params := extractPaginationParams(c)

	var filters []shared.Filter

	if name := c.Query("name"); name != "" {
		filters = append(filters, shared.Filter{
			Field:    "name",
			Operator: shared.OperatorILike,
			Value:    "%" + name + "%",
		})
	}
	if email := c.Query("email"); email != "" {
		filters = append(filters, shared.Filter{
			Field:    "email",
			Operator: shared.OperatorILike,
			Value:    "%" + email + "%",
		})
	}

	result, err := h.userUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := dto.ToPaginatedResponse(result, dto.ToUserResponse)
	response.Success(c, http.StatusOK, "Users retrieved successfully", resp)
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Get current authenticated user profile details
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.ApiResponse{data=dto.UserResponse} "User profile retrieved successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      404  {object}  response.ApiResponse "Not Found - User not found"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := extractCurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user, err := h.userUseCase.Find(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Profile retrieved successfully", dto.ToUserResponse(user))
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update current authenticated user profile details (e.g. name)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.UpdateProfileRequest true "User profile update details"
// @Success      200  {object}  response.ApiResponse{data=dto.UserResponse} "Profile updated successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      404  {object}  response.ApiResponse "Not Found - User not found"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid request payload"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := extractCurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdateProfileRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user := &domain.User{
		ID:   userID,
		Name: req.Name,
	}

	if err := h.userUseCase.UpdateProfile(c.Request.Context(), user); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", dto.ToUserResponse(user))
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Update user information by UUID. Requires 'update_user' permission.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                    true  "User UUID" format(uuid)
// @Param        request  body      dto.UpdateProfileRequest  true  "User update details"
// @Success      200      {object}  response.ApiResponse{data=dto.UserResponse} "User updated successfully"
// @Failure      401      {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403      {object}  response.ApiResponse "Forbidden - Requires update_user permission"
// @Failure      404      {object}  response.ApiResponse "Not Found - User not found"
// @Failure      422      {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format or request payload"
// @Failure      500      {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdateProfileRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user := &domain.User{
		ID:   userID,
		Name: req.Name,
	}

	if err := h.userUseCase.UpdateProfile(c.Request.Context(), user); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "User updated successfully", dto.ToUserResponse(user))
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Delete a user by UUID. Requires 'delete_user' permission.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User UUID" format(uuid)
// @Success      200  {object}  response.ApiResponse "User deleted successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403  {object}  response.ApiResponse "Forbidden - Requires delete_user permission"
// @Failure      404  {object}  response.ApiResponse "Not Found - User not found"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.userUseCase.Delete(c.Request.Context(), userID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "User deleted successfully", nil)
}

// AddRoles godoc
// @Summary      Add roles to user
// @Description  Assign one or more roles to a user by role UUIDs. Requires 'add_user_role' permission.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                      true  "User UUID" format(uuid)
// @Param        request  body      dto.UpdateUserRolesRequest  true  "Role UUIDs to assign"
// @Success      200      {object}  response.ApiResponse "Roles added to user successfully"
// @Failure      401      {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403      {object}  response.ApiResponse "Forbidden - Requires add_user_role permission"
// @Failure      404      {object}  response.ApiResponse "Not Found - User not found"
// @Failure      422      {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format or request payload"
// @Failure      500      {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/{id}/roles [post]
func (h *UserHandler) AddRoles(c *gin.Context) {
	userID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdateUserRolesRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.userUseCase.AddRoles(c.Request.Context(), userID, req.RoleIDs); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Roles added to user successfully", nil)
}

// RemoveRoles godoc
// @Summary      Remove roles from user
// @Description  Remove one or more assigned roles from a user by role UUIDs. Requires 'remove_user_role' permission.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                      true  "User UUID" format(uuid)
// @Param        request  body      dto.UpdateUserRolesRequest  true  "Role UUIDs to remove"
// @Success      200      {object}  response.ApiResponse "Roles removed from user successfully"
// @Failure      401      {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403      {object}  response.ApiResponse "Forbidden - Requires remove_user_role permission"
// @Failure      404      {object}  response.ApiResponse "Not Found - User not found"
// @Failure      422      {object}  response.ApiResponse "Unprocessable Entity - Invalid UUID format or request payload"
// @Failure      500      {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/users/{id}/roles [delete]
func (h *UserHandler) RemoveRoles(c *gin.Context) {
	userID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.UpdateUserRolesRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.userUseCase.RemoveRoles(c.Request.Context(), userID, req.RoleIDs); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Roles removed from user successfully", nil)
}
