package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain"
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
// @Description  Retrieve detailed information of a user by UUID
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "User UUID" format(uuid)
// @Success      200  {object}  middleware.ApiResponse{data=dto.UserWithRoleResponse}
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      404  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
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

	middleware.Success(c, http.StatusOK, "User retrieved successfully", dto.ToUserWithRoleResponse(user))
}

// ListUsers godoc
// @Summary      List users
// @Description  Get paginated list of users with optional filters and sorting
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int     false  "Page number (default: 1)"
// @Param        limit  query     int     false  "Items per page (default: 10, max: 100)"
// @Param        sort   query     string  false  "Sorting criteria (e.g. name:asc, created_at:desc or -created_at)"
// @Param        name   query     string  false  "Filter by user name (partial match)"
// @Param        email  query     string  false  "Filter by user email (partial match)"
// @Success      200    {object}  middleware.ApiResponse{data=dto.PaginatedUserResponse}
// @Failure      401    {object}  middleware.ApiResponse
// @Failure      500    {object}  middleware.ApiResponse
// @Router       /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	params := extractPaginationParams(c)

	var filters []domain.Filter

	if name := c.Query("name"); name != "" {
		filters = append(filters, domain.Filter{
			Field:    "name",
			Operator: domain.OperatorILike,
			Value:    "%" + name + "%",
		})
	}
	if email := c.Query("email"); email != "" {
		filters = append(filters, domain.Filter{
			Field:    "email",
			Operator: domain.OperatorILike,
			Value:    "%" + email + "%",
		})
	}

	result, err := h.userUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := dto.ToPaginatedResponse(result, dto.ToUserResponse)
	middleware.Success(c, http.StatusOK, "Users retrieved successfully", response)
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Get current authenticated user profile details
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  middleware.ApiResponse{data=dto.UserResponse}
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      404  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
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

	middleware.Success(c, http.StatusOK, "Profile retrieved successfully", dto.ToUserResponse(user))
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update current authenticated user profile information
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.UpdateProfileRequest true "User profile update details"
// @Success      200  {object}  middleware.ApiResponse{data=dto.UserResponse}
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
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

	middleware.Success(c, http.StatusOK, "Profile updated successfully", dto.ToUserResponse(user))
}

// AddRoles godoc
// @Summary      Add roles to user
// @Description  Assign one or more roles to a user by role UUIDs
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                      true  "User UUID" format(uuid)
// @Param        request  body      dto.UpdateUserRolesRequest  true  "Role UUIDs to assign"
// @Success      200      {object}  middleware.ApiResponse
// @Failure      401      {object}  middleware.ApiResponse
// @Failure      404      {object}  middleware.ApiResponse
// @Failure      422      {object}  middleware.ApiResponse
// @Failure      500      {object}  middleware.ApiResponse
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

	middleware.Success(c, http.StatusOK, "Roles added to user successfully", nil)
}

// RemoveRoles godoc
// @Summary      Remove roles from user
// @Description  Remove one or more assigned roles from a user by role UUIDs
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                      true  "User UUID" format(uuid)
// @Param        request  body      dto.UpdateUserRolesRequest  true  "Role UUIDs to remove"
// @Success      200      {object}  middleware.ApiResponse
// @Failure      401      {object}  middleware.ApiResponse
// @Failure      404      {object}  middleware.ApiResponse
// @Failure      422      {object}  middleware.ApiResponse
// @Failure      500      {object}  middleware.ApiResponse
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

	middleware.Success(c, http.StatusOK, "Roles removed from user successfully", nil)
}
