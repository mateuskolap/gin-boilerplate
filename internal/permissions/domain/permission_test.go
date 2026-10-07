package domain

import "testing"

func TestAllPermissionsIncludesOrganizationPermissions(t *testing.T) {
	want := []PermissionName{
		PermissionViewOrganization,
		PermissionCreateOrganization,
		PermissionUpdateOrganization,
		PermissionDeleteOrganization,
	}
	for _, expected := range want {
		found := false
		for _, permission := range AllPermissions {
			if permission == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AllPermissions does not include %q", expected)
		}
	}
}
