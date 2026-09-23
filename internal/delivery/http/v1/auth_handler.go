package v1

import (
	"errors"
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUseCase domain.AuthUseCase
}

func NewAuthHandler(authUseCase domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with name, email and password. Assigns default User role.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "User registration details"
// @Success      201  {object}  response.ApiResponse{data=dto.UserResponse} "User registered successfully"
// @Failure      409  {object}  response.ApiResponse "Conflict - Email already in use"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid payload validation"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
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

	if err := h.authUseCase.Register(c.Request.Context(), user); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully", dto.ToUserResponse(user))
}

// Login godoc
// @Summary      User authentication
// @Description  Authenticate user with email and password, returning JWT access token and refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login credentials"
// @Success      200  {object}  response.ApiResponse{data=dto.LoginResponse} "Login successful with token pair"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Invalid email or password"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid payload validation"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req, err := bindJSON[dto.LoginRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	authTokens, err := h.authUseCase.Login(
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

	response.Success(c, http.StatusOK, "Login successful", dto.LoginResponse{
		AccessToken:  authTokens.AccessToken,
		RefreshToken: authTokens.RefreshToken,
	})
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a newly issued access token and rotated refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshRequest true "Refresh token payload"
// @Success      200  {object}  response.ApiResponse{data=dto.LoginResponse} "Tokens refreshed successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Invalid, expired or revoked refresh token"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid payload validation"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	req, err := bindJSON[dto.RefreshRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	authTokens, err := h.authUseCase.Refresh(
		c.Request.Context(),
		req.RefreshToken,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Token refreshed successfully", dto.LoginResponse{
		AccessToken:  authTokens.AccessToken,
		RefreshToken: authTokens.RefreshToken,
	})
}

// Logout godoc
// @Summary      User logout
// @Description  Revoke access token by adding it to the blacklist and optionally revoke the refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.LogoutRequest false "Optional refresh token to revoke alongside access token"
// @Success      200  {object}  response.ApiResponse "Logged out successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		_ = c.Error(shared.NewAppError(
			shared.ErrTypeValidation,
			"Validation failed",
			err,
		))
		return
	}

	token, _ := extractToken(c)

	if err := h.authUseCase.Logout(c.Request.Context(), token, req.RefreshToken); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}
