package v1

import (
	"gin-boilerplate/internal/domain/shared"
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
		return uuid.Nil(), shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Unauthorized",
			nil,
		)
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil(), shared.NewAppError(
			shared.ErrTypeValidation,
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
		return uuid.Nil(), shared.NewAppError(
			shared.ErrTypeValidation,
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
		return req, shared.NewAppError(
			shared.ErrTypeValidation,
			"Validation failed",
			err,
		)
	}
	return req, nil
}

func extractToken(c *gin.Context) (string, error) {
	tokenString, exists := c.Get("raw_token")
	if !exists {
		return "", shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Token not found in request context",
			nil,
		)
	}
	return tokenString.(string), nil
}

// extractPaginationParams extracts page, limit, and sort parameters from query parameters.
func extractPaginationParams(c *gin.Context) shared.PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	params := shared.PaginationParams{
		Page:  page,
		Limit: limit,
	}

	sortQuery := c.Query("sort")
	if sortQuery != "" {
		sortFields := strings.Split(sortQuery, ",")

		params.Sort = make([]shared.SortParam, 0, len(sortFields))

		for _, s := range sortFields {
			s = strings.TrimSpace(s)

			if s == "" {
				continue
			}

			field, direction := parseSortOption(s)

			if field != "" {
				params.Sort = append(params.Sort, shared.SortParam{
					Field:     field,
					Direction: direction,
				})
			}
		}
	}

	params.Sanitize()
	return params
}

func parseSortOption(s string) (field string, direction shared.SortDirection) {
	if strings.HasPrefix(s, "-") {
		field := strings.TrimSpace(strings.TrimPrefix(s, "-"))
		return field, shared.SortDesc
	}

	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		field := strings.TrimSpace(parts[0])
		dir := strings.ToLower(strings.TrimSpace(parts[1]))
		if dir == "desc" {
			return field, shared.SortDesc
		}
		return field, shared.SortAsc
	}

	return s, shared.SortAsc
}
