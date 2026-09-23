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
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil(), shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Unauthorized",
			nil,
		)
	}

	userIDString, ok := value.(string)
	if !ok {
		return uuid.Nil(), shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Unauthorized",
			nil,
		)
	}
	userID, err := uuid.Parse(userIDString)
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
	token, ok := tokenString.(string)
	if !ok || token == "" {
		return "", shared.NewAppError(
			shared.ErrTypeUnauthorized,
			"Token not found in request context",
			nil,
		)
	}
	return token, nil
}

// extractPaginationParams extracts page, limit, and sort parameters from query parameters.
func extractPaginationParams(c *gin.Context) (shared.PaginationParams, error) {
	page, err := parseBoundedPositiveInt(c.DefaultQuery("page", "1"), "page", 0)
	if err != nil {
		return shared.PaginationParams{}, err
	}
	limit, err := parseBoundedPositiveInt(c.DefaultQuery("limit", "10"), "limit", 100)
	if err != nil {
		return shared.PaginationParams{}, err
	}

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
				return shared.PaginationParams{}, invalidQueryParameter("sort")
			}

			field, direction, err := parseSortOption(s)
			if err != nil {
				return shared.PaginationParams{}, err
			}
			params.Sort = append(params.Sort, shared.SortParam{
				Field:     field,
				Direction: direction,
			})
		}
	}

	return params, nil
}

func parseSortOption(s string) (field string, direction shared.SortDirection, err error) {
	if strings.HasPrefix(s, "-") {
		field := strings.TrimSpace(strings.TrimPrefix(s, "-"))
		if field == "" {
			return "", "", invalidQueryParameter("sort")
		}
		return field, shared.SortDesc, nil
	}

	if strings.Contains(s, ":") {
		parts := strings.SplitN(s, ":", 2)
		field := strings.TrimSpace(parts[0])
		dir := strings.ToLower(strings.TrimSpace(parts[1]))
		if field == "" || (dir != "asc" && dir != "desc") {
			return "", "", invalidQueryParameter("sort")
		}
		if dir == "desc" {
			return field, shared.SortDesc, nil
		}
		return field, shared.SortAsc, nil
	}

	if strings.TrimSpace(s) == "" {
		return "", "", invalidQueryParameter("sort")
	}
	return strings.TrimSpace(s), shared.SortAsc, nil
}

func parseBoundedPositiveInt(value, name string, maximum int) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 || (maximum > 0 && parsed > maximum) {
		return 0, invalidQueryParameter(name)
	}
	return parsed, nil
}

func invalidQueryParameter(name string) error {
	return shared.NewAppError(
		shared.ErrTypeValidation,
		"Invalid query parameter: "+name,
		nil,
	)
}
