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
	if !strings.Contains(string(model), "ID: entity.ID") || strings.Contains(string(model), "BaseModel:") {
		t.Fatalf("generated model should initialize promoted fields directly: %s", model)
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
