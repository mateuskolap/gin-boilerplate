package v1

import (
	"gin-boilerplate/internal/domain"
	"net/http"
	"strconv"
	"strings"

	"uuid"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes = 2 * 1024 * 1024

func extractCurrentUserID(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil(), domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Unauthorized",
			nil,
		)
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil(), domain.NewAppError(
			domain.ErrTypeValidation,
			"Invalid user ID format",
			err,
		)
	}

	return userID, nil
}

func extractParamID(c *gin.Context, paramName string) (uuid.UUID, error) {
	idStr := c.Param(paramName)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil(), domain.NewAppError(
			domain.ErrTypeValidation,
			"Invalid ID format",
			err,
		)
	}
	return id, nil
}

func bindJSON[T any](c *gin.Context) (T, error) {
	var req T
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, domain.NewAppError(
			domain.ErrTypeValidation,
			"Invalid request payload: "+err.Error(),
			err,
		)
	}
	return req, nil
}

func extractToken(c *gin.Context) (string, error) {
	tokenString, exists := c.Get("raw_token")
	if !exists {
		return "", domain.NewAppError(
			domain.ErrTypeUnauthorized,
			"Token not found in request context",
			nil,
		)
	}
	return tokenString.(string), nil
}

// extractPaginationParams extracts page, limit, and sort parameters from query parameters.
func extractPaginationParams(c *gin.Context) domain.PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	params := domain.PaginationParams{
		Page:  page,
		Limit: limit,
	}

	sortQuery := c.Query("sort")
	if sortQuery != "" {
		sortFields := strings.Split(sortQuery, ",")

		params.Sort = make([]domain.SortParam, 0, len(sortFields))

		for _, s := range sortFields {
			s = strings.TrimSpace(s)

			if s == "" {
				continue
			}

			field, direction := parseSortOption(s)

			if field != "" {
				params.Sort = append(params.Sort, domain.SortParam{
					Field:     field,
					Direction: direction,
				})
			}
		}
	}

	params.Sanitize()
	return params
}

func parseSortOption(s string) (field string, direction domain.SortDirection) {
	if strings.HasPrefix(s, "-") {
		field := strings.TrimSpace(strings.TrimPrefix(s, "-"))
		return field, domain.SortDesc
	}

	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		field := strings.TrimSpace(parts[0])
		dir := strings.ToLower(strings.TrimSpace(parts[1]))
		if dir == "desc" {
			return field, domain.SortDesc
		}
		return field, domain.SortAsc
	}

	return s, domain.SortAsc
}
