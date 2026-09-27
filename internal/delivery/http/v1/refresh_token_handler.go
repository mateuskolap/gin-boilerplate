package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RefreshTokenHandler struct {
	refreshTokenUseCase domain.RefreshTokenUseCase
}

func NewRefreshTokenHandler(refreshTokenUseCase domain.RefreshTokenUseCase) *RefreshTokenHandler {
	return &RefreshTokenHandler{
		refreshTokenUseCase: refreshTokenUseCase,
	}
}

// ListRefreshTokensByAuthUser godoc
// @Summary      List active sessions
// @Description  Get paginated list of active refresh tokens (sessions) for the authenticated user
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        page        query     int     false  "Page number (default: 1)" minimum(1)
// @Param        limit       query     int     false  "Items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param        sort        query     string  false  "Sorting criteria (e.g. created_at:desc or -created_at)"
// @Param        user_agent  query     string  false  "Filter by user agent (partial match)"
// @Param        ip_address  query     string  false  "Filter by IP address (partial match)"
// @Success      200         {object}  response.ApiResponse{data=dto.PaginatedRefreshTokenResponse} "Sessions retrieved successfully"
// @Failure      401         {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      422         {object}  response.ApiResponse "Unprocessable Entity - Invalid filter or sorting parameter"
// @Failure      429         {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500         {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/sessions [get]
func (h *RefreshTokenHandler) ListRefreshTokensByAuthUser(c *gin.Context) {
	userID, err := extractCurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	params, err := extractPaginationParams(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var filters []shared.Filter

	if userAgent := c.Query("user_agent"); userAgent != "" {
		filters = append(filters, shared.Filter{
			Field:    "user_agent",
			Operator: shared.OperatorILike,
			Value:    "%" + userAgent + "%",
		})
	}
	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filters = append(filters, shared.Filter{
			Field:    "ip_address",
			Operator: shared.OperatorILike,
			Value:    "%" + ipAddress + "%",
		})
	}

	result, err := h.refreshTokenUseCase.ListActiveByUserID(c.Request.Context(), userID, params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := dto.ToPaginatedResponse(result, dto.ToRefreshTokenResponse)
	response.Success(c, http.StatusOK, "Refresh tokens retrieved successfully", resp)
}

// RevokeSession godoc
// @Summary      End a session
// @Description  Revoke one active refresh token session owned by the authenticated user. Existing access tokens remain valid until expiry.
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Session UUID" format(uuid)
// @Success      204  {object}  nil "Session ended"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      404  {object}  response.ApiResponse "Not Found - Active session not found"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid session UUID"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/sessions/{id} [delete]
func (h *RefreshTokenHandler) RevokeSession(c *gin.Context) {
	userID, err := extractCurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	sessionID, err := extractParamID(c, "id")
	if err != nil {
		_ = c.Error(err)
		return
	}

	if err := h.refreshTokenUseCase.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokeOtherSessions godoc
// @Summary      End other sessions
// @Description  Revoke every active session except the session identified by the current refresh token. Existing access tokens remain valid until expiry.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.RevokeOtherSessionsRequest true "Current refresh token"
// @Success      200  {object}  response.ApiResponse{data=dto.RevokeOtherSessionsResponse} "Other sessions ended"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Invalid current refresh token"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid request payload"
// @Failure      429  {object}  response.ApiResponse "Too Many Requests - Rate limit exceeded"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/auth/sessions/revoke-others [post]
func (h *RefreshTokenHandler) RevokeOtherSessions(c *gin.Context) {
	userID, err := extractCurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	req, err := bindJSON[dto.RevokeOtherSessionsRequest](c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	revokedSessions, err := h.refreshTokenUseCase.RevokeOtherSessions(c.Request.Context(), userID, req.CurrentRefreshToken)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Other sessions ended", dto.RevokeOtherSessionsResponse{
		RevokedSessions: revokedSessions,
	})
}
