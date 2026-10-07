package postgres

import (
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	organizationpostgres "gin-boilerplate/internal/organizations/adapters/postgres"
	userdomain "gin-boilerplate/internal/users/domain"
	"testing"
	"uuid"
)

func TestUserModelMapsOptionalOrganization(t *testing.T) {
	model := userModelFromDomain(&userdomain.User{})
	if model.OrganizationID != nil {
		t.Fatalf("user without organization mapped to organization ID %v", model.OrganizationID)
	}
	if user := userDomainFromModel(model); user.OrganizationID != uuid.Nil() || user.Organization != nil {
		t.Fatalf("user without organization mapped to domain value %+v", user)
	}
}

func TestUserModelMapsPreloadedOrganization(t *testing.T) {
	organizationID := uuid.New()
	model := &UserModel{
		OrganizationID: &organizationID,
		Organization: &organizationpostgres.OrganizationModel{
			BaseModel: postgresinfra.BaseModel{ID: organizationID},
			Name:      "Example organization",
		},
	}

	user := userDomainFromModel(model)
	if user.OrganizationID != organizationID || user.Organization == nil || user.Organization.ID != organizationID || user.Organization.Name != "Example organization" {
		t.Fatalf("preloaded organization mapping = %+v", user)
	}
}
