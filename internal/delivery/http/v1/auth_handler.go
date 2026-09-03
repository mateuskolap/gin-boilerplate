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

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user account with name, email and password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "User registration details"
// @Success      201  {object}  middleware.ApiResponse{data=dto.UserProfileResponse}
// @Failure      400  {object}  middleware.ApiResponse
// @Failure      409  {object}  middleware.ApiResponse
// @Failure      422  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	req, err := bindJSON[dto.RegisterRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user := &domain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userUseCase.Register(c.Request.Context(), user); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusCreated, "User registered successfully", dto.ToUserResponse(user))
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user credentials and return a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "User login credentials"
// @Success      200  {object}  middleware.ApiResponse{data=dto.LoginResponse}
// @Failure      400  {object}  middleware.ApiResponse
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req, err := bindJSON[dto.LoginRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	authTokens, err := h.userUseCase.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Login successful", dto.LoginResponse{
		AccessToken:  authTokens.AccessToken,
		RefreshToken: authTokens.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	req, err := bindJSON[dto.RefreshRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	authTokens, err := h.userUseCase.Refresh(
		c.Request.Context(),
		req.RefreshToken,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Token refreshed successfully", dto.LoginResponse{
		AccessToken:  authTokens.AccessToken,
		RefreshToken: authTokens.RefreshToken,
	})
}

// Logout godoc
// @Summary      User logout
// @Description  Invalidate current user JWT token
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  middleware.ApiResponse
// @Failure      401  {object}  middleware.ApiResponse
// @Failure      500  {object}  middleware.ApiResponse
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshRequest
	_ = c.ShouldBindJSON(&req)

	token, err := extractToken(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.userUseCase.Logout(c.Request.Context(), token, req.RefreshToken); err != nil {
		_ = c.Error(err)
		return
	}

	middleware.Success(c, http.StatusOK, "Logged out successfully", nil)
}
