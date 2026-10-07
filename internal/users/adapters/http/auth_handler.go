package httpadapter

import (
	"errors"
	"gin-boilerplate/internal/delivery/http/authcookie"
	"gin-boilerplate/internal/delivery/http/common"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain/shared"
	userdto "gin-boilerplate/internal/users/adapters/http/dto"
	userdomain "gin-boilerplate/internal/users/domain"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authUseCase    userdomain.AuthUseCase
	useCookies     bool
	secureCookies  bool
	cookieSameSite http.SameSite
	accessTTL      time.Duration
	refreshTTL     time.Duration
}

func NewAuthHandler(authUseCase userdomain.AuthUseCase, useCookies, secureCookies bool, accessTTL, refreshTTL time.Duration, cookieSameSite http.SameSite) *AuthHandler {
	return &AuthHandler{
		authUseCase:    authUseCase,
		useCookies:     useCookies,
		secureCookies:  secureCookies,
		cookieSameSite: cookieSameSite,
		accessTTL:      accessTTL,
		refreshTTL:     refreshTTL,
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with name, email and password. An organization UUID is optional. Assigns the default User role.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.RegisterRequest true "User registration details"
// @Success      201  {object}  response.ApiResponse{data=userdto.UserResponse} "User registered successfully"
// @Failure      409  {object}  response.ApiResponse "Conflict - Email already in use"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid payload validation"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	req, err := common.BindJSON[userdto.RegisterRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user := &userdomain.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}
	if req.OrganizationID != nil {
		user.OrganizationID = *req.OrganizationID
	}

	if err := h.authUseCase.Register(c.Request.Context(), user); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully", userdto.ToUserResponse(user))
}

// Login godoc
// @Summary      User authentication
// @Description  Authenticate with email and password. Tokens are returned in the response body or set as HttpOnly cookies according to AUTH_TOKEN_TRANSPORT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.LoginRequest true "Login credentials"
// @Success      200  {object}  response.ApiResponse{data=userdto.LoginResponse} "Login successful; token data is omitted in cookie mode"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Invalid email or password"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid payload validation"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	req, err := common.BindJSON[userdto.LoginRequest](c)
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

	h.respondWithTokens(c, "Login successful", authTokens)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a valid refresh token for a new token pair. The refresh token is read from the body or cookie according to AUTH_TOKEN_TRANSPORT; the response uses the same transport.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body userdto.RefreshRequest false "Refresh token payload in body mode; omitted in cookie mode"
// @Success      200  {object}  response.ApiResponse{data=userdto.LoginResponse} "Token data is omitted in cookie mode"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Invalid, expired or revoked refresh token"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid payload validation"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken := authcookie.Get(c, authcookie.RefreshTokenName)
	if !h.useCookies {
		req, err := common.BindJSON[userdto.RefreshRequest](c)
		if err != nil {
			_ = c.Error(err)
			return
		}
		refreshToken = req.RefreshToken
	}

	authTokens, err := h.authUseCase.Refresh(
		c.Request.Context(),
		refreshToken,
		c.ClientIP(),
		c.Request.UserAgent(),
	)
	if err != nil {
		_ = c.Error(err)
		return
	}

	h.respondWithTokens(c, "Token refreshed successfully", authTokens)
}

// Logout godoc
// @Summary      User logout
// @Description  Revoke access token by adding it to the blacklist and optionally revoke the refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body userdto.LogoutRequest false "Refresh token in body mode; cookie mode reads it from the HttpOnly cookie"
// @Success      200  {object}  response.ApiResponse "Logged out successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken := authcookie.Get(c, authcookie.RefreshTokenName)
	if !h.useCookies {
		var req userdto.LogoutRequest
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, common.MaxBodyBytes)
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			_ = c.Error(shared.NewAppError(
				shared.ErrTypeValidation,
				"Validation failed",
				err,
			))
			return
		}
		refreshToken = req.RefreshToken
	}

	token, _ := common.Token(c)

	if err := h.authUseCase.Logout(c.Request.Context(), token, refreshToken); err != nil {
		_ = c.Error(err)
		return
	}

	if h.useCookies {
		authcookie.Clear(c, h.secureCookies, h.cookieSameSite)
	}
	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

// ChangePassword godoc
// @Summary      Change password
// @Description  Verify the current password, set a new password, and invalidate all user sessions and access tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body userdto.ChangePasswordRequest true "Current and new password"
// @Success      204  {object}  nil "Password changed; authenticate again with the new password"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Current password is invalid"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid password payload"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/password [patch]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, err := common.CurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := common.BindJSON[userdto.ChangePasswordRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.authUseCase.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		_ = c.Error(err)
		return
	}

	if h.useCookies {
		authcookie.Clear(c, h.secureCookies, h.cookieSameSite)
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) respondWithTokens(c *gin.Context, message string, tokens *userdomain.AuthTokens) {
	c.Header("Cache-Control", "no-store")
	if h.useCookies {
		authcookie.Set(c, tokens.AccessToken, tokens.RefreshToken, h.accessTTL, h.refreshTTL, h.secureCookies, h.cookieSameSite)
		response.Success(c, http.StatusOK, message, nil)
		return
	}
	response.Success(c, http.StatusOK, message, userdto.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}
