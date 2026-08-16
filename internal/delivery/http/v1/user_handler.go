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

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	user, err := h.userUseCase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	middleware.Success(c, http.StatusOK, "Profile retrieved successfully", dto.ToUserResponse(user))
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	req, err := bindJSON[dto.UpdateProfileRequest](c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	user := &domain.User{
		Name: req.Name,
	}
	user.ID = userID

	if err := h.userUseCase.UpdateProfile(c.Request.Context(), user); err != nil {
		middleware.HandleError(c, err)
		return
	}

	middleware.Success(c, http.StatusOK, "Profile updated successfully", nil)
}
