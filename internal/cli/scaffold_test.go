package cli

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDomainNames(t *testing.T) {
	tests := []struct {
		name, stem, feature string
	}{
		{"User", "user", "users"},
		{"ActivityLog", "activityLog", "activity_logs"},
		{"Category", "category", "categories"},
		{"APIKey", "apiKey", "api_keys"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, stem, _, feature, err := domainNames(test.name)
			if err != nil {
				t.Fatal(err)
			}
			if stem != test.stem || feature != test.feature {
				t.Fatalf("domainNames(%q) = (%q, %q), want (%q, %q)", test.name, stem, feature, test.stem, test.feature)
			}
		})
	}
}

func TestMakeDomainCreatesFormattedGoFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}

	feature, err := makeDomain(root, "ActivityLog")
	if err != nil {
		t.Fatal(err)
	}
	if feature != "activity_logs" {
		t.Fatalf("feature = %q, want activity_logs", feature)
	}

	base := filepath.Join(root, "internal", feature)
	files := []string{
		"domain/activity_log.go",
		"application/activity_log.go",
		"adapters/postgres/activity_log_model.go",
		"adapters/postgres/activity_log_repository.go",
	}
	for _, name := range files {
		path := filepath.Join(base, name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated file %s: %v", name, err)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.AllErrors); err != nil {
			t.Fatalf("generated invalid Go file %s: %v", name, err)
		}
	}

	model, err := os.ReadFile(filepath.Join(base, files[2]))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(model), "BaseModel: postgresinfra.BaseModelFromDomain(entity.BaseModel)") ||
		!strings.Contains(string(model), "BaseModel: model.BaseModel.ToDomain()") {
		t.Fatalf("generated model should use the shared base model conversion: %s", model)
	}
	if strings.Contains(string(model), "DeletedAt") {
		t.Fatalf("generated model should omit soft-delete support by default: %s", model)
	}

	migrations, err := filepath.Glob(filepath.Join(root, migrationsDirectory, "*_create_"+feature+".up.sql"))
	if err != nil || len(migrations) != 1 {
		t.Fatalf("generated up migration paths=%v error=%v", migrations, err)
	}
	upSQL, err := os.ReadFile(migrations[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(upSQL), "CREATE TABLE activity_logs") || strings.Contains(string(upSQL), "deleted_at") {
		t.Fatalf("default migration SQL = %s", upSQL)
	}
}

func TestMakeDomainWithSoftDeleteGeneratesColumnAndMappings(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}

	feature, err := makeDomainWithSoftDelete(root, "Invoice", true)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, "internal", feature)
	domain, err := os.ReadFile(filepath.Join(base, "domain", "invoice.go"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(base, "adapters", "postgres", "invoice_model.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range [][]byte{domain, model} {
		if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, parser.AllErrors); err != nil {
			t.Fatalf("generated invalid soft-delete Go file: %v", err)
		}
	}
	if !strings.Contains(string(domain), "shared.BaseSoftDeleteModel") {
		t.Fatalf("generated domain does not embed soft-delete base model: %s", domain)
	}
	for _, fragment := range []string{
		"DeletedAt gorm.DeletedAt `gorm:\"index\"`",
		"model.DeletedAt = gorm.DeletedAt{Time: *entity.DeletedAt, Valid: true}",
		"entity.DeletedAt = &deletedAt",
	} {
		if !strings.Contains(string(model), fragment) {
			t.Fatalf("generated model missing %q: %s", fragment, model)
		}
	}

	migrations, err := filepath.Glob(filepath.Join(root, migrationsDirectory, "*_create_"+feature+".up.sql"))
	if err != nil || len(migrations) != 1 {
		t.Fatalf("generated up migration paths=%v error=%v", migrations, err)
	}
	upSQL, err := os.ReadFile(migrations[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"deleted_at TIMESTAMPTZ", "idx_invoices_deleted_at ON invoices (deleted_at)"} {
		if !strings.Contains(string(upSQL), fragment) {
			t.Fatalf("generated migration missing %q: %s", fragment, upSQL)
		}
	}
	downSQL, err := os.ReadFile(strings.TrimSuffix(migrations[0], ".up.sql") + ".down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if string(downSQL) != "DROP TABLE invoices;\n" {
		t.Fatalf("down migration = %q", downSQL)
	}
}

func TestMakeDomainWithActivityLogsGeneratesAuditScaffold(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}

	feature, err := makeDomainWithOptions(root, "Invoice", true, true)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, "internal", feature)
	paths := []string{
		filepath.Join(base, "domain", "invoice.go"),
		filepath.Join(base, "adapters", "postgres", "invoice_model.go"),
		filepath.Join(base, "adapters", "postgres", "invoice_repository.go"),
	}
	contents := make([]string, len(paths))
	for i, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		contents[i] = string(source)
		if _, err := parser.ParseFile(token.NewFileSet(), path, source, parser.AllErrors); err != nil {
			t.Fatalf("generated invalid Go file %s: %v", path, err)
		}
	}
	for _, fragment := range []string{
		"shared.BaseSoftDeleteModel",
		"ActivityLogSubjectType() activitylogdomain.ActivitySubjectType",
		"ActivityLogID()",
		"activity:\"track\"",
	} {
		if !strings.Contains(contents[0], fragment) {
			t.Fatalf("generated domain missing %q: %s", fragment, contents[0])
		}
	}
	if !strings.Contains(contents[1], "DeletedAt gorm.DeletedAt") {
		t.Fatalf("generated model missing soft-delete support: %s", contents[1])
	}
	for _, fragment := range []string{
		"activitylogdomain.ActivityLogRepository",
		"postgresinfra.NewActivityLoggingRepository",
	} {
		if !strings.Contains(contents[2], fragment) {
			t.Fatalf("generated repository missing %q: %s", fragment, contents[2])
		}
	}
}

func TestMakeDomainRejectsInvalidAndExistingDomains(t *testing.T) {
	root := t.TempDir()
	internal := filepath.Join(root, "internal")
	if err := os.Mkdir(internal, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := makeDomain(root, "user-profile"); err == nil {
		t.Fatal("expected invalid domain name to fail")
	}

	existing := filepath.Join(internal, "users")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(existing, "keep.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := makeDomain(root, "User"); err == nil {
		t.Fatal("expected existing domain to fail")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("existing domain was modified: %v", err)
	}
}
