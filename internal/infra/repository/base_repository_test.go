package repository

import (
	"context"
	"strings"
	"testing"

	"gin-boilerplate/internal/domain"
	"gin-boilerplate/internal/domain/shared"

	"uuid"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func dryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=postgres dbname=test sslmode=disable",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}
	return db
}

func TestApplyFiltersBuildsSafeQueriesAndRejectsInvalidInput(t *testing.T) {
	db := dryRunDB(t)
	query, err := applyFilters(db.Model(&domain.User{}), []shared.Filter{
		{Field: "name", Operator: shared.OperatorILike, Value: "%alice%"},
		{Field: "id", Operator: shared.OperatorIn, Value: []int{1, 2}},
		{Field: "deleted_at", Operator: shared.OperatorIsNull},
	})
	if err != nil {
		t.Fatalf("applyFilters() error = %v", err)
	}
	statement := query.Find(&[]domain.User{}).Statement.SQL.String()
	for _, fragment := range []string{`"name" ILIKE $1`, `"id" IN ($2,$3)`, `"deleted_at" IS NULL`} {
		if !strings.Contains(statement, fragment) {
			t.Errorf("query %q does not contain %q", statement, fragment)
		}
	}

	for _, filter := range []shared.Filter{
		{Field: "name; DROP TABLE users", Operator: shared.OperatorEquals},
		{Field: "name", Operator: "= OR 1=1"},
	} {
		if _, err := applyFilters(db.Model(&domain.User{}), []shared.Filter{filter}); err == nil {
			t.Errorf("applyFilters() accepted unsafe filter %+v", filter)
		}
	}
}

func TestIdentifierPathValidation(t *testing.T) {
	for _, value := range []string{"name", "_private", "Roles.CreatedAt"} {
		if !isSafeIdentifierPath(value) {
			t.Errorf("isSafeIdentifierPath(%q) = false", value)
		}
	}
	for _, value := range []string{"", "1name", "Roles..Name", "name; DROP", `name"`} {
		if isSafeIdentifierPath(value) {
			t.Errorf("isSafeIdentifierPath(%q) = true", value)
		}
	}
}

func TestBaseRepositoryUsesTransactionFromContext(t *testing.T) {
	db, txDB := dryRunDB(t), dryRunDB(t)
	repo := newBaseRepository[domain.User](db)
	ctx := context.WithValue(context.Background(), txKey{}, txDB)
	if got := repo.getDB(ctx); got != txDB {
		t.Fatal("repository did not use the transaction stored in context")
	}

	plainContext := context.Background()
	if got := repo.getDB(plainContext); got.Statement.Context != plainContext {
		t.Fatal("repository did not attach the request context to the default database")
	}
}

func TestNewModelActivityTracksOnlyTaggedFields(t *testing.T) {
	actorID, requestIP := uuid.New(), "192.0.2.1"
	ctx := shared.WithRequestIP(shared.WithActorID(context.Background(), actorID), requestIP)
	user := &domain.User{Name: "Alice", Email: "alice@example.com", Password: "secret"}
	user.ID = uuid.New()
	activity := newModelActivity(ctx, user, "created", nil)
	if activity == nil {
		t.Fatal("activity was not created")
	}
	if activity.Event != domain.ActivityEvent("user.created") || activity.ActorID == nil || *activity.ActorID != actorID || activity.IPAddress == nil || *activity.IPAddress != requestIP {
		t.Fatalf("activity=%+v", activity)
	}
	attributes := activity.Changes["attributes"].(map[string]any)["new"].(map[string]any)
	if attributes["name"] != "Alice" || attributes["email"] != "alice@example.com" || attributes["password"] != nil {
		t.Fatalf("logged attributes=%+v", attributes)
	}
}
