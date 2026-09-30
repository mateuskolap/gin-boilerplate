package postgres

import (
	"context"
	"strings"
	"testing"

	activitylogdomain "gin-boilerplate/internal/activity_logs/domain"
	"gin-boilerplate/internal/domain/shared"
	userdomain "gin-boilerplate/internal/users/domain"

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

type testUserModel struct {
	ID        uuid.UUID
	Name      string
	DeletedAt gorm.DeletedAt
}

func TestApplyFiltersBuildsSafeQueriesAndRejectsInvalidInput(t *testing.T) {
	db := dryRunDB(t)
	query, err := applyFilters(db.Model(&testUserModel{}), []shared.Filter{
		{Field: "name", Operator: shared.OperatorILike, Value: "%alice%"},
		{Field: "id", Operator: shared.OperatorIn, Value: []int{1, 2}},
		{Field: "deleted_at", Operator: shared.OperatorIsNull},
	})
	if err != nil {
		t.Fatalf("applyFilters() error = %v", err)
	}
	statement := query.Find(&[]testUserModel{}).Statement.SQL.String()
	for _, fragment := range []string{`"name" ILIKE $1`, `"id" IN ($2,$3)`, `"deleted_at" IS NULL`} {
		if !strings.Contains(statement, fragment) {
			t.Errorf("query %q does not contain %q", statement, fragment)
		}
	}

	for _, filter := range []shared.Filter{
		{Field: "name; DROP TABLE users", Operator: shared.OperatorEquals},
		{Field: "name", Operator: "= OR 1=1"},
	} {
		if _, err := applyFilters(db.Model(&testUserModel{}), []shared.Filter{filter}); err == nil {
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
	repo := &BaseRepository[int]{db: db}
	ctx := context.WithValue(context.Background(), txKey{}, txDB)
	if got := repo.DB(ctx); got != txDB {
		t.Fatal("repository did not use the transaction stored in context")
	}

	plainContext := context.Background()
	if got := repo.DB(plainContext); got.Statement.Context != plainContext {
		t.Fatal("repository did not attach the request context to the default database")
	}
}

func TestNewModelActivityTracksOnlyTaggedFields(t *testing.T) {
	actorID, requestIP := uuid.New(), "192.0.2.1"
	ctx := shared.WithRequestIP(shared.WithActorID(context.Background(), actorID), requestIP)
	user := &userdomain.User{Name: "Alice", Email: "alice@example.com", Password: "secret"}
	user.ID = uuid.New()
	activity := newModelActivity(ctx, user, "created", nil)
	if activity == nil {
		t.Fatal("activity was not created")
	}
	if activity.Event != activitylogdomain.ActivityEvent("user.created") || activity.ActorID == nil || *activity.ActorID != actorID || activity.IPAddress == nil || *activity.IPAddress != requestIP {
		t.Fatalf("activity=%+v", activity)
	}
	attributes := activity.Changes["attributes"].(map[string]any)["new"].(map[string]any)
	if attributes["name"] != "Alice" || attributes["email"] != "alice@example.com" || attributes["password"] != nil {
		t.Fatalf("logged attributes=%+v", attributes)
	}
}
