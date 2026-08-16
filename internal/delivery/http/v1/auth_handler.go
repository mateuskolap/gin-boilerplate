package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	userUseCase domain.UserUseCase
}

func NewAuthHandler(userUseCase domain.UserUseCase) *AuthHandler {
	return &AuthHandler{
		userUseCase: userUseCase,
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	req, err := bindJSON[dto.RegisterRequest](c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userUseCase.Register(c.Request.Context(), user); err != nil {
		middleware.HandleError(c, err)
		return
	}

	middleware.Success(c, http.StatusCreated, "User registered successfully", dto.ToUserResponse(user))
}

func (h *AuthHandler) Login(c *gin.Context) {
	req, err := bindJSON[dto.LoginRequest](c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	token, err := h.userUseCase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	middleware.Success(c, http.StatusOK, "Login successful", dto.LoginResponse{Token: token})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, err := extractToken(c)
	if err != nil {
		middleware.HandleError(c, err)
		return
	}

	if err := h.userUseCase.Logout(c.Request.Context(), token); err != nil {
		middleware.HandleError(c, err)
		return
	}

	middleware.Success(c, http.StatusOK, "Logged out successfully", nil)
}
