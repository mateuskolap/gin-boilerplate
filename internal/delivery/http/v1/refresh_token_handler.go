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
