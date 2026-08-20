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
	userID, err := extractUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user, err := h.userUseCase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Profile retrieved successfully", dto.ToUserProfileResponse(user))
}

// UpdateProfile godoc
// @Summary      Update user profile
// @Description  Update current authenticated user profile information
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.UpdateProfileRequest true "User profile update details"
// @Success      200  {object}  middleware.ApiResponse
// @Failure      400  {object}  middleware.ApiResponse
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := extractUserID(c)
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
		Name: req.Name,
	}
	user.ID = userID

	if err := h.userUseCase.UpdateProfile(c.Request.Context(), user); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Profile updated successfully", nil)
}
