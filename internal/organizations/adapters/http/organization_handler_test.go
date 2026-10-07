package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"gin-boilerplate/internal/domain/shared"
	organizationdomain "gin-boilerplate/internal/organizations/domain"

	"github.com/gin-gonic/gin"
)

type organizationHandlerUseCase struct {
	organizationdomain.OrganizationUseCase
	organization *organizationdomain.Organization
	listResult   *shared.PaginatedResult[organizationdomain.Organization]
	findID       uuid.UUID
	filters      []shared.Filter
	created      *organizationdomain.Organization
	updated      *organizationdomain.Organization
	deletedID    uuid.UUID
}

func (f *organizationHandlerUseCase) Find(_ context.Context, id uuid.UUID) (*organizationdomain.Organization, error) {
	f.findID = id
	return f.organization, nil
}

func (f *organizationHandlerUseCase) List(_ context.Context, _ shared.PaginationParams, filters []shared.Filter) (*shared.PaginatedResult[organizationdomain.Organization], error) {
	f.filters = filters
	return f.listResult, nil
}

func (f *organizationHandlerUseCase) Create(_ context.Context, organization *organizationdomain.Organization) error {
	f.created = organization
	return nil
}

func (f *organizationHandlerUseCase) Update(_ context.Context, organization *organizationdomain.Organization) error {
	f.updated = organization
	return nil
}

func (f *organizationHandlerUseCase) Delete(_ context.Context, id uuid.UUID) error {
	f.deletedID = id
	return nil
}

func serveOrganizationHandler(t *testing.T, method, route, target, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, route, handler)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestOrganizationHandlerCRUDAndListFilter(t *testing.T) {
	organizationID := uuid.New()
	fake := &organizationHandlerUseCase{
		organization: &organizationdomain.Organization{BaseSoftDeleteModel: shared.BaseSoftDeleteModel{BaseModel: shared.BaseModel{ID: organizationID}}, Name: "Acme"},
		listResult:   &shared.PaginatedResult[organizationdomain.Organization]{Page: 1, Limit: 10},
	}
	handler := NewOrganizationHandler(fake)

	list := serveOrganizationHandler(t, http.MethodGet, "/organizations", "/organizations?name=Acme", "", handler.ListOrganizations)
	if list.Code != http.StatusOK || len(fake.filters) != 1 || fake.filters[0].Field != "name" || fake.filters[0].Value != "%Acme%" {
		t.Fatalf("ListOrganizations() status=%d filters=%+v body=%s", list.Code, fake.filters, list.Body.String())
	}

	create := serveOrganizationHandler(t, http.MethodPost, "/organizations", "/organizations", `{"name":"New Org"}`, handler.CreateOrganization)
	if create.Code != http.StatusCreated || fake.created == nil || fake.created.Name != "New Org" {
		t.Fatalf("CreateOrganization() status=%d organization=%+v body=%s", create.Code, fake.created, create.Body.String())
	}

	find := serveOrganizationHandler(t, http.MethodGet, "/organizations/:id", "/organizations/"+organizationID.String(), "", handler.FindOrganization)
	if find.Code != http.StatusOK || fake.findID != organizationID {
		t.Fatalf("FindOrganization() status=%d id=%v body=%s", find.Code, fake.findID, find.Body.String())
	}

	update := serveOrganizationHandler(t, http.MethodPut, "/organizations/:id", "/organizations/"+organizationID.String(), `{"name":"Acme Updated"}`, handler.UpdateOrganization)
	if update.Code != http.StatusOK || fake.updated == nil || fake.updated.ID != organizationID || fake.updated.Name != "Acme Updated" {
		t.Fatalf("UpdateOrganization() status=%d organization=%+v body=%s", update.Code, fake.updated, update.Body.String())
	}

	delete := serveOrganizationHandler(t, http.MethodDelete, "/organizations/:id", "/organizations/"+organizationID.String(), "", handler.DeleteOrganization)
	if delete.Code != http.StatusNoContent || fake.deletedID != organizationID {
		t.Fatalf("DeleteOrganization() status=%d id=%v body=%s", delete.Code, fake.deletedID, delete.Body.String())
	}
}
