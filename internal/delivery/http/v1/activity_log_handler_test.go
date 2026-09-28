package v1

import (
	"net/http"
	"testing"
	"time"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

	"uuid"
)

func TestActivityLogHandlerListsWithFilters(t *testing.T) {
	actorID, subjectID := uuid.New(), uuid.New()
	requestIP := "192.0.2.1"
	activity := &domain.ActivityLog{
		ID:          uuid.New(),
		Event:       domain.ActivityUserUpdated,
		ActorID:     &actorID,
		SubjectType: domain.ActivitySubjectUser,
		SubjectID:   subjectID,
		Changes:     map[string]any{"attributes": map[string]any{}},
		IPAddress:   &requestIP,
		CreatedAt:   time.Now().UTC(),
	}
	fake := &httpActivityLogUseCase{result: &shared.PaginatedResult[domain.ActivityLog]{Page: 1, Limit: 10, Items: []*domain.ActivityLog{activity}}}
	handler := NewActivityLogHandler(fake)
	target := "/activity-logs?event=user.updated&subject_type=user&subject_id=" + subjectID.String() + "&actor_id=" + actorID.String() + "&created_from=2026-01-01T00:00:00Z&created_to=2026-01-02T00:00:00Z"
	recorder := serveHTTPHandler(t, http.MethodGet, "/activity-logs", target, "", handler.ListActivityLogs)

	if recorder.Code != http.StatusOK || fake.calls != 1 || len(fake.filters) != 6 {
		t.Fatalf("status=%d calls=%d filters=%+v body=%s", recorder.Code, fake.calls, fake.filters, recorder.Body.String())
	}
	if len(fake.params.Sort) != 1 || fake.params.Sort[0] != (shared.SortParam{Field: "created_at", Direction: shared.SortDesc}) {
		t.Fatalf("default sort=%+v", fake.params.Sort)
	}
	if response := decodeResponse(t, recorder); response.Success != true {
		t.Fatalf("response=%+v", response)
	}
}

func TestActivityLogHandlerRejectsInvalidFilters(t *testing.T) {
	fake := &httpActivityLogUseCase{}
	recorder := serveHTTPHandler(t, http.MethodGet, "/activity-logs", "/activity-logs?actor_id=invalid", "", NewActivityLogHandler(fake).ListActivityLogs)
	if recorder.Code != http.StatusUnprocessableEntity || fake.calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", recorder.Code, fake.calls, recorder.Body.String())
	}
}
