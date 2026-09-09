package v1

import (
	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/middleware"
	"gin-boilerplate/internal/domain"
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

func (h *RefreshTokenHandler) ListRefreshTokensByAuthUser(c *gin.Context) {
	userID, err := extractCurrentUserID(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	params := extractPaginationParams(c)

	var filters []domain.Filter

	if userAgent := c.Query("user_agent"); userAgent != "" {
		filters = append(filters, domain.Filter{
			Field:    "user_agent",
			Operator: domain.OperatorILike,
			Value:    "%" + userAgent + "%",
		})
	}
	if ipAddress := c.Query("ip_address"); ipAddress != "" {
		filters = append(filters, domain.Filter{
			Field:    "ip_address",
			Operator: domain.OperatorILike,
			Value:    "%" + ipAddress + "%",
		})
	}

	result, err := h.refreshTokenUseCase.ListByUserID(c.Request.Context(), userID, params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response := dto.ToPaginatedResponse(result, dto.ToRefreshTokenResponse)
	middleware.Success(c, http.StatusOK, "Refresh tokens retrieved successfully", response)
}
