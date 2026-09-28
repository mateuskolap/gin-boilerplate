package v1

import (
	"net/http"
	"time"

	"gin-boilerplate/internal/delivery/http/dto"
	"gin-boilerplate/internal/delivery/http/response"
	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

	"uuid"

	"github.com/gin-gonic/gin"
)

type ActivityLogHandler struct {
	activityLogUseCase domain.ActivityLogUseCase
}

func NewActivityLogHandler(activityLogUseCase domain.ActivityLogUseCase) *ActivityLogHandler {
	return &ActivityLogHandler{activityLogUseCase: activityLogUseCase}
}

// ListActivityLogs godoc
// @Summary      List activity logs
// @Description  Retrieve paginated administrative activity logs. Requires 'view_activity_log' permission.
// @Tags         Activity logs
// @Produce      json
// @Security     BearerAuth
// @Param        page          query     int     false  "Page number (default: 1)" minimum(1)
// @Param        limit         query     int     false  "Items per page (default: 10, max: 100)" minimum(1) maximum(100)
// @Param        sort          query     string  false  "Sorting criteria (default: created_at:desc)"
// @Param        event         query     string  false  "Filter by event"
// @Param        subject_type  query     string  false  "Filter by subject type"
// @Param        subject_id    query     string  false  "Filter by subject UUID" format(uuid)
// @Param        actor_id      query     string  false  "Filter by actor UUID" format(uuid)
// @Param        request_id    query     string  false  "Filter by request UUID" format(uuid)
// @Param        created_from  query     string  false  "Filter from RFC3339 timestamp"
// @Param        created_to    query     string  false  "Filter until RFC3339 timestamp"
// @Success      200  {object}  response.ApiResponse{data=dto.PaginatedActivityLogResponse} "Activity logs retrieved successfully"
// @Failure      401  {object}  response.ApiResponse "Unauthorized - Missing or invalid token"
// @Failure      403  {object}  response.ApiResponse "Forbidden - Requires view_activity_log permission"
// @Failure      422  {object}  response.ApiResponse "Unprocessable Entity - Invalid filter or sorting parameter"
// @Failure      500  {object}  response.ApiResponse "Internal server error"
// @Router       /api/v1/activity-logs [get]
func (h *ActivityLogHandler) ListActivityLogs(c *gin.Context) {
	params, err := extractPaginationParams(c)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if len(params.Sort) == 0 {
		params.Sort = []shared.SortParam{{Field: "created_at", Direction: shared.SortDesc}}
	}

	filters, err := activityLogFilters(c)
	if err != nil {
		_ = c.Error(err)
		return
	}

	result, err := h.activityLogUseCase.List(c.Request.Context(), params, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, http.StatusOK, "Activity logs retrieved successfully", dto.ToPaginatedResponse(result, dto.ToActivityLogResponse))
}

func activityLogFilters(c *gin.Context) ([]shared.Filter, error) {
	filters := make([]shared.Filter, 0, 6)
	for _, field := range []string{"event", "subject_type"} {
		if value := c.Query(field); value != "" {
			filters = append(filters, shared.Filter{Field: field, Operator: shared.OperatorEquals, Value: value})
		}
	}
	for _, field := range []string{"subject_id", "actor_id", "request_id"} {
		if value := c.Query(field); value != "" {
			id, err := uuid.Parse(value)
			if err != nil {
				return nil, invalidQueryParameter(field)
			}
			filters = append(filters, shared.Filter{Field: field, Operator: shared.OperatorEquals, Value: id})
		}
	}
	for _, dateFilter := range []struct {
		query    string
		operator shared.FilterOperator
	}{
		{query: "created_from", operator: shared.OperatorGreaterThanOrEqual},
		{query: "created_to", operator: shared.OperatorLessThanOrEqual},
	} {
		if value := c.Query(dateFilter.query); value != "" {
			createdAt, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return nil, invalidQueryParameter(dateFilter.query)
			}
			filters = append(filters, shared.Filter{Field: "created_at", Operator: dateFilter.operator, Value: createdAt})
		}
	}
	return filters, nil
}
