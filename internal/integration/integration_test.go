package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"gin-boilerplate/config"
	activitylogpostgres "gin-boilerplate/internal/activity_logs/adapters/postgres"
	"gin-boilerplate/internal/bootstrap"
	"gin-boilerplate/internal/domain/port"
	"gin-boilerplate/internal/domain/shared"
	postgresinfra "gin-boilerplate/internal/infra/postgres"
	"gin-boilerplate/internal/infra/queue"
	"gin-boilerplate/internal/infra/ratelimit"
	"gin-boilerplate/internal/infra/repository"
	permissionpostgres "gin-boilerplate/internal/permissions/adapters/postgres"
	permissiondomain "gin-boilerplate/internal/permissions/domain"
	refreshpostgres "gin-boilerplate/internal/refresh_tokens/adapters/postgres"
	refreshtokendomain "gin-boilerplate/internal/refresh_tokens/domain"
	rolepostgres "gin-boilerplate/internal/roles/adapters/postgres"
	roledomain "gin-boilerplate/internal/roles/domain"
	userpostgres "gin-boilerplate/internal/users/adapters/postgres"
	userdomain "gin-boilerplate/internal/users/domain"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"uuid"
)

func testPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	db := testPostgresSchema(t)
	if err := db.AutoMigrate(&userpostgres.UserModel{}, &rolepostgres.RoleModel{}, &permissionpostgres.PermissionModel{}, &refreshpostgres.RefreshTokenModel{}, &activitylogpostgres.ActivityLogModel{}); err != nil {
		t.Fatalf("migrate isolated schema: %v", err)
	}
	return db
}

func testPostgresSchema(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}

	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		t.Fatalf("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	if !isTestDatabaseName(parsed.Path) {
		t.Fatalf("TEST_DATABASE_URL database must be named 'test' or end in '_test' or '-test'")
	}

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL test connection: %v", err)
	}
	schema := "integration_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	var db *gorm.DB
	schemaCreated := false
	t.Cleanup(func() {
		if db != nil {
			if sqlDB, err := db.DB(); err == nil {
				_ = sqlDB.Close()
			}
		}
		if schemaCreated {
			if err := admin.Exec("DROP SCHEMA \"" + schema + "\" CASCADE").Error; err != nil {
				t.Errorf("drop isolated schema: %v", err)
			}
		}
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	schemaCreated = true

	schemaURL := *parsed
	query := schemaURL.Query()
	query.Set("search_path", schema)
	schemaURL.RawQuery = query.Encode()
	db, err = gorm.Open(postgres.Open(schemaURL.String()), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open schema-scoped connection: %v", err)
	}
	return db
}

func testRedis(t *testing.T) (*redis.Client, int, int) {
	t.Helper()
	address := os.Getenv("TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("set TEST_REDIS_ADDR to a disposable Redis instance")
	}
	redisDB, err := requiredRedisDB(t, "TEST_REDIS_DB")
	if err != nil {
		t.Fatal(err)
	}
	queueDB, err := requiredRedisDB(t, "TEST_QUEUE_REDIS_DB")
	if err != nil {
		t.Fatal(err)
	}
	if redisDB == queueDB {
		t.Fatal("TEST_REDIS_DB and TEST_QUEUE_REDIS_DB must differ")
	}
	client := redis.NewClient(&redis.Options{Addr: address, DB: redisDB})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("connect to Redis test instance: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client, redisDB, queueDB
}

func requiredRedisDB(t *testing.T, key string) (int, error) {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		return 0, fmt.Errorf("set %s to a disposable Redis database number", key)
	}
	db, err := strconv.Atoi(value)
	if err != nil || db < 0 || db > 15 {
		return 0, fmt.Errorf("%s must be a Redis database number from 0 to 15", key)
	}
	return db, nil
}

func isTestDatabaseName(path string) bool {
	name := strings.ToLower(strings.Trim(path, "/"))
	return name == "test" || strings.HasSuffix(name, "_test") || strings.HasSuffix(name, "-test")
}

func TestRepositoriesAndTransactionsAgainstPostgres(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	activityLogs := activitylogpostgres.NewActivityLogRepository(db)
	users := userpostgres.NewUserRepository(db, activityLogs)
	roles := rolepostgres.NewRoleRepository(db, activityLogs)
	authorization := permissionpostgres.NewAuthorizationRepository(db)

	alice := &userdomain.User{Name: "Alice", Email: "alice@example.test", Password: "hash"}
	bob := &userdomain.User{Name: "Bob", Email: "bob@example.test", Password: "hash"}
	for _, user := range []*userdomain.User{alice, bob} {
		if err := users.Create(ctx, user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}

	role := &roledomain.Role{Name: "operator"}
	permission := &permissiondomain.Permission{Name: "integration.permission"}
	if err := roles.Create(ctx, role); err != nil {
		t.Fatalf("create role: %v", err)
	}
	permissionModel := permissionpostgres.PermissionModel{Name: permission.Name}
	if err := db.WithContext(ctx).Create(&permissionModel).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	permission.ID = permissionModel.ID
	if found, err := permissionpostgres.NewPermissionRepository(db).List(ctx, shared.PaginationParams{Page: 1, Limit: 10}, nil); err != nil || len(found.Items) != 1 {
		t.Fatalf("list permissions after create: %#v, %v", found, err)
	}
	if err := users.AddRoles(ctx, alice.ID, []uuid.UUID{role.ID}); err != nil {
		t.Fatalf("assign user role: %v", err)
	}
	if err := roles.AddPermissions(ctx, role.ID, []uuid.UUID{permission.ID}); err != nil {
		t.Fatalf("assign role permission: %v", err)
	}
	if err := users.AddRoles(ctx, alice.ID, nil); err != nil {
		t.Fatalf("assign empty role list: %v", err)
	}
	if err := roles.AddPermissions(ctx, role.ID, nil); err != nil {
		t.Fatalf("assign empty permission list: %v", err)
	}
	if allowed, err := authorization.UserHasPermission(ctx, alice.ID, permissiondomain.PermissionName(permission.Name)); err != nil || !allowed {
		t.Fatalf("UserHasPermission() = %v, %v; want true, nil", allowed, err)
	}
	if allowed, err := authorization.UserHasPermission(ctx, bob.ID, permissiondomain.PermissionName(permission.Name)); err != nil || allowed {
		t.Fatalf("permission for unrelated user = %v, %v; want false, nil", allowed, err)
	}

	loaded, err := roles.GetByIDWithPermissions(ctx, role.ID)
	if err != nil || loaded == nil || len(loaded.Permissions) != 1 {
		t.Fatalf("GetByName with permissions = %#v, %v", loaded, err)
	}
	if err := roles.RemovePermissions(ctx, role.ID, []uuid.UUID{permission.ID}); err != nil {
		t.Fatalf("remove role permission: %v", err)
	}
	if allowed, err := authorization.UserHasPermission(ctx, alice.ID, permissiondomain.PermissionName(permission.Name)); err != nil || allowed {
		t.Fatalf("permission after removal = %v, %v; want false, nil", allowed, err)
	}
	if err := roles.AddPermissions(ctx, role.ID, []uuid.UUID{permission.ID}); err != nil {
		t.Fatalf("restore role permission: %v", err)
	}
	if err := users.RemoveRoles(ctx, alice.ID, []uuid.UUID{role.ID}); err != nil {
		t.Fatalf("remove user role: %v", err)
	}
	if allowed, err := authorization.UserHasPermission(ctx, alice.ID, permissiondomain.PermissionName(permission.Name)); err != nil || allowed {
		t.Fatalf("permission after role removal = %v, %v; want false, nil", allowed, err)
	}
	if err := users.AddRoles(ctx, alice.ID, []uuid.UUID{role.ID}); err != nil {
		t.Fatalf("restore user role: %v", err)
	}

	found, err := users.GetByEmail(ctx, "alice@example.test")
	if err != nil || found == nil || found.ID != alice.ID {
		t.Fatalf("GetByEmail() = %#v, %v", found, err)
	}
	if withRoles, err := users.GetByEmailWithRoles(ctx, alice.Email); err != nil || withRoles == nil || len(withRoles.Roles) != 1 {
		t.Fatalf("GetByEmailWithRoles() = %#v, %v", withRoles, err)
	}
	alice.Name = "Alice Updated"
	if err := users.Update(ctx, alice); err != nil {
		t.Fatalf("update user: %v", err)
	}
	page, err := users.List(ctx, shared.PaginationParams{Page: 1, Limit: 1, Sort: []shared.SortParam{{Field: "email", Direction: shared.SortDesc}}}, []shared.Filter{{Field: "email", Operator: shared.OperatorILike, Value: "%alice%"}})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].Name != "Alice Updated" {
		t.Fatalf("filtered user list = %#v, %v", page, err)
	}
	if err := users.Delete(ctx, alice.ID); err != nil {
		t.Fatalf("soft delete user: %v", err)
	}
	if deleted, err := users.GetByID(ctx, alice.ID); err != nil || deleted != nil {
		t.Fatalf("GetByID after soft delete = %#v, %v; want nil, nil", deleted, err)
	}

	transactions := postgresinfra.NewGormTransactionManagerRepository(db)
	rollbackUser := &userdomain.User{Name: "Rolled Back", Email: "rollback@example.test", Password: "hash"}
	rollbackErr := errors.New("abort transaction")
	if err := transactions.Do(ctx, func(txCtx context.Context) error {
		if err := users.Create(txCtx, rollbackUser); err != nil {
			return err
		}
		return rollbackErr
	}); !errors.Is(err, rollbackErr) {
		t.Fatalf("transaction error = %v, want rollback marker", err)
	}
	if rolledBack, err := users.GetByEmail(ctx, rollbackUser.Email); err != nil || rolledBack != nil {
		t.Fatalf("rolled back user = %#v, %v; want nil, nil", rolledBack, err)
	}
	committedUser := &userdomain.User{Name: "Committed", Email: "committed@example.test", Password: "hash"}
	if err := transactions.Do(ctx, func(txCtx context.Context) error { return users.Create(txCtx, committedUser) }); err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if committed, err := users.GetByEmail(ctx, committedUser.Email); err != nil || committed == nil {
		t.Fatalf("committed user = %#v, %v", committed, err)
	}
}

func TestRefreshTokenRepositoryStateTransitionsAgainstPostgres(t *testing.T) {
	db := testPostgres(t)
	ctx := context.Background()
	users := userpostgres.NewUserRepository(db, activitylogpostgres.NewActivityLogRepository(db))
	tokens := refreshpostgres.NewRefreshTokenRepository(db)
	owner := &userdomain.User{Name: "Owner", Email: "owner@example.test", Password: "hash"}
	other := &userdomain.User{Name: "Other", Email: "other@example.test", Password: "hash"}
	if err := users.Create(ctx, owner); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	keep := &refreshtokendomain.RefreshToken{UserID: owner.ID, TokenHash: "keep", ExpiresAt: now.Add(time.Hour), IpAddress: "127.0.0.1", UserAgent: "test"}
	rotate := &refreshtokendomain.RefreshToken{UserID: owner.ID, TokenHash: "rotate", ExpiresAt: now.Add(time.Hour), IpAddress: "127.0.0.1", UserAgent: "test"}
	expired := &refreshtokendomain.RefreshToken{UserID: owner.ID, TokenHash: "expired", ExpiresAt: now.Add(-time.Hour), IpAddress: "127.0.0.1", UserAgent: "test"}
	foreign := &refreshtokendomain.RefreshToken{UserID: other.ID, TokenHash: "foreign", ExpiresAt: now.Add(time.Hour), IpAddress: "127.0.0.1", UserAgent: "test"}
	for _, token := range []*refreshtokendomain.RefreshToken{keep, rotate, expired, foreign} {
		if err := tokens.Create(ctx, token); err != nil {
			t.Fatalf("create token %q: %v", token.TokenHash, err)
		}
	}

	byHash, err := tokens.FindByTokenHash(ctx, keep.TokenHash)
	if err != nil || byHash == nil || byHash.ID != keep.ID {
		t.Fatalf("FindByTokenHash() = %#v, %v", byHash, err)
	}
	listed, err := tokens.ListByUserID(ctx, owner.ID, shared.PaginationParams{Page: 1, Limit: 10}, []shared.Filter{{Field: "user_id", Operator: shared.OperatorEquals, Value: other.ID}})
	if err != nil || listed.Total != 3 || len(listed.Items) != 3 {
		t.Fatalf("ListByUserID() = %#v, %v; caller user filter must not escape scope", listed, err)
	}
	byRelatedUser, err := tokens.ListByUserID(ctx, owner.ID, shared.PaginationParams{Page: 1, Limit: 10}, []shared.Filter{
		{Field: "User.email", Operator: shared.OperatorEquals, Value: owner.Email},
		{Field: "User.name", Operator: shared.OperatorEquals, Value: owner.Name},
	})
	if err != nil || byRelatedUser.Total != 3 {
		t.Fatalf("ListByUserID() related user filter = %#v, %v", byRelatedUser, err)
	}
	if revoked, err := tokens.Revoke(ctx, keep.ID, other.ID, nil); err != nil || revoked {
		t.Fatalf("revoke from another user = %v, %v; want false, nil", revoked, err)
	}
	if revoked, err := tokens.Revoke(ctx, expired.ID, owner.ID, nil); err != nil || revoked {
		t.Fatalf("revoke expired token = %v, %v; want false, nil", revoked, err)
	}
	replacementID := uuid.New()
	if revoked, err := tokens.Revoke(ctx, rotate.ID, owner.ID, &replacementID); err != nil || !revoked {
		t.Fatalf("revoke active token = %v, %v; want true, nil", revoked, err)
	}
	revokedToken, err := tokens.FindByTokenHash(ctx, rotate.TokenHash)
	if err != nil || revokedToken == nil || revokedToken.RevokedAt == nil || revokedToken.ReplacedBy == nil || *revokedToken.ReplacedBy != replacementID {
		t.Fatalf("revoked token state = %#v, %v", revokedToken, err)
	}
	if revoked, err := tokens.Revoke(ctx, rotate.ID, owner.ID, nil); err != nil || revoked {
		t.Fatalf("revoke already revoked token = %v, %v; want false, nil", revoked, err)
	}

	if err := tokens.RevokeAllExcept(ctx, owner.ID, keep.ID); err != nil {
		t.Fatalf("RevokeAllExcept(): %v", err)
	}
	keepAfter, err := tokens.FindByTokenHash(ctx, keep.TokenHash)
	if err != nil || keepAfter == nil || keepAfter.RevokedAt != nil {
		t.Fatalf("excepted token state = %#v, %v", keepAfter, err)
	}
	if err := tokens.RevokeAllByUserID(ctx, owner.ID); err != nil {
		t.Fatalf("RevokeAllByUserID(): %v", err)
	}
	keepAfter, err = tokens.FindByTokenHash(ctx, keep.TokenHash)
	if err != nil || keepAfter == nil || keepAfter.RevokedAt == nil {
		t.Fatalf("all-revoked token state = %#v, %v", keepAfter, err)
	}
	deleted, err := tokens.DeleteExpiredBefore(ctx, now)
	if err != nil || deleted != 1 {
		t.Fatalf("DeleteExpiredBefore() = %d, %v; want one row, nil", deleted, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if revoked, err := tokens.Revoke(canceled, keep.ID, owner.ID, nil); err == nil || revoked {
		t.Fatalf("Revoke() canceled call = %v, %v; want false and error", revoked, err)
	}
	if _, err := tokens.DeleteExpiredBefore(canceled, now); err == nil {
		t.Fatal("DeleteExpiredBefore() ignored canceled context")
	}
}

func TestDatabaseSeederIsRepeatableAndTransactional(t *testing.T) {
	db := testPostgres(t)
	cfg := &config.Config{AdminName: "Integration Admin", AdminEmail: "admin@example.test", AdminPassword: "integration-password"}
	if err := db.Create(&permissionpostgres.PermissionModel{Name: "obsolete.integration.permission"}).Error; err != nil {
		t.Fatal(err)
	}
	seed := &bootstrap.Application{Config: cfg, DB: db}
	for range 2 {
		if err := seed.Seed(context.Background()); err != nil {
			t.Fatalf("Application.Seed(): %v", err)
		}
	}

	var permissionCount int64
	if err := db.Model(&permissionpostgres.PermissionModel{}).Count(&permissionCount).Error; err != nil || permissionCount != int64(len(permissiondomain.AllPermissions)) {
		t.Fatalf("permission count = %d, %v; want %d", permissionCount, err, len(permissiondomain.AllPermissions))
	}
	var obsoleteCount int64
	if err := db.Model(&permissionpostgres.PermissionModel{}).Where("name = ?", "obsolete.integration.permission").Count(&obsoleteCount).Error; err != nil || obsoleteCount != 0 {
		t.Fatalf("obsolete permission count = %d, %v; want zero", obsoleteCount, err)
	}
	admin, err := userpostgres.NewUserRepository(db, activitylogpostgres.NewActivityLogRepository(db)).GetByEmailWithRoles(context.Background(), cfg.AdminEmail)
	if err != nil || admin == nil || len(admin.Roles) != 1 || admin.Roles[0].Name != roledomain.RoleAdmin {
		t.Fatalf("seeded admin = %#v, %v", admin, err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(cfg.AdminPassword)); err != nil {
		t.Fatalf("seeded admin password is not hashed correctly: %v", err)
	}
	var adminRole rolepostgres.RoleModel
	if err := db.Preload("Permissions").Where("name = ?", roledomain.RoleAdmin).First(&adminRole).Error; err != nil || len(adminRole.Permissions) != len(permissiondomain.AllPermissions) {
		t.Fatalf("admin role permissions = %d, %v; want %d", len(adminRole.Permissions), err, len(permissiondomain.AllPermissions))
	}

	rollbackDB := testPostgres(t)
	invalid := &config.Config{AdminName: "Invalid Admin", AdminEmail: "bad@example.test", AdminPassword: strings.Repeat("x", 73)}
	if err := (&bootstrap.Application{Config: invalid, DB: rollbackDB}).Seed(context.Background()); err == nil {
		t.Fatal("Application.Seed() succeeded despite an invalid admin password")
	}
	for _, model := range []any{&permissionpostgres.PermissionModel{}, &rolepostgres.RoleModel{}, &userpostgres.UserModel{}} {
		var count int64
		if err := rollbackDB.Model(model).Count(&count).Error; err != nil || count != 0 {
			t.Errorf("failed seeding left rows in %T: count=%d err=%v", model, count, err)
		}
	}
}

func TestRedisCacheRateLimitAndQueueWorker(t *testing.T) {
	client, _, queueDB := testRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	namespace := "integration:" + strings.ReplaceAll(uuid.New().String(), "-", "")

	cache := repository.NewRedisCache(client)
	cacheKey := namespace + ":cache"
	t.Cleanup(func() { _ = client.Del(context.Background(), cacheKey, namespace+":bad-json").Err() })
	if err := cache.Set(ctx, cacheKey, map[string]any{"value": "cached"}, time.Minute); err != nil {
		t.Fatalf("cache Set(): %v", err)
	}
	var cached struct {
		Value string `json:"value"`
	}
	if err := cache.Get(ctx, cacheKey, &cached); err != nil || cached.Value != "cached" {
		t.Fatalf("cache Get() = %#v, %v", cached, err)
	}
	missing := "unchanged"
	if err := cache.Get(ctx, namespace+":missing", &missing); err != nil || missing != "unchanged" {
		t.Fatalf("cache miss changed destination: %q, %v", missing, err)
	}
	if err := client.Set(ctx, namespace+":bad-json", "{", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if err := cache.Get(ctx, namespace+":bad-json", &cached); err == nil {
		t.Fatal("cache Get() accepted malformed JSON")
	}

	limitKey := namespace + ":rate"
	t.Cleanup(func() { _ = client.Del(context.Background(), limitKey).Err() })
	limiter := ratelimit.NewRedisLimiter(client)
	first, err := limiter.Allow(ctx, limitKey, 1, time.Minute)
	if err != nil || !first.Allowed || first.Remaining != 0 {
		t.Fatalf("first rate-limit request = %#v, %v", first, err)
	}
	second, err := limiter.Allow(ctx, limitKey, 1, time.Minute)
	if err != nil || second.Allowed || second.RetryAfter <= 0 {
		t.Fatalf("second rate-limit request = %#v, %v", second, err)
	}

	queueClient := redis.NewClient(&redis.Options{Addr: os.Getenv("TEST_REDIS_ADDR"), DB: queueDB})
	t.Cleanup(func() { _ = queueClient.Close() })
	handler := &integrationHandler{taskType: namespace + ".task", done: make(chan string, 1)}
	worker, err := queue.NewWorker(queueClient, queue.WorkerConfig{Concurrency: 1, ShutdownTimeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), handler)
	if err != nil {
		t.Fatalf("create queue worker: %v", err)
	}
	if err := worker.Start(); err != nil {
		t.Fatalf("start queue worker: %v", err)
	}
	t.Cleanup(worker.Shutdown)
	dispatcher := queue.NewDispatcher(queueClient)
	info, err := dispatcher.Dispatch(ctx, port.QueueTask{Type: handler.taskType, Payload: []byte(`{"value":"queued"}`)}, port.DispatchOptions{Timeout: time.Minute, MaxRetries: 0})
	if err != nil {
		t.Fatalf("dispatch queue task: %v", err)
	}
	inspector := asynq.NewInspectorFromRedisClient(queueClient)
	t.Cleanup(func() { _ = inspector.DeleteTask(info.Queue, info.ID) })
	select {
	case payload := <-handler.done:
		if payload != `{"value":"queued"}` {
			t.Fatalf("queue handler payload = %s", payload)
		}
	case <-ctx.Done():
		t.Fatalf("queue task was not handled: %v", ctx.Err())
	}
}

type integrationHandler struct {
	taskType string
	done     chan string
}

func (h *integrationHandler) TaskType() string { return h.taskType }

func (h *integrationHandler) HandleTask(_ context.Context, payload json.RawMessage) error {
	h.done <- string(payload)
	return nil
}

func testApplicationConfig(t *testing.T) *config.Config {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	if !isTestDatabaseName(parsed.Path) {
		t.Fatal("TEST_DATABASE_URL database must be named 'test' or end in '_test' or '-test'")
	}
	user := parsed.User
	if user == nil {
		t.Fatal("TEST_DATABASE_URL must include credentials")
	}
	password, _ := user.Password()
	dbPort, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("TEST_DATABASE_URL must include a port: %v", err)
	}
	address := os.Getenv("TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("set TEST_REDIS_ADDR to a disposable Redis instance")
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf("invalid TEST_REDIS_ADDR: %v", err)
	}
	redisPort, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("invalid Redis port: %v", err)
	}
	redisDB, err := requiredRedisDB(t, "TEST_REDIS_DB")
	if err != nil {
		t.Fatal(err)
	}
	queueDB, err := requiredRedisDB(t, "TEST_QUEUE_REDIS_DB")
	if err != nil || redisDB == queueDB {
		t.Fatalf("invalid test Redis DB pair: %v", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve test HTTP port: %v", err)
	}
	httpPort := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	cfg := &config.Config{
		Env:                "test",
		DBHost:             parsed.Hostname(),
		DBPort:             dbPort,
		DBUser:             user.Username(),
		DBPassword:         password,
		DBName:             strings.Trim(parsed.Path, "/"),
		DBSSLMode:          "disable",
		AuthTokenTransport: "body", Port: httpPort,
		DBMaxOpenConnections: 5, DBMaxIdleConnections: 1, DBConnectionLifetime: time.Minute, DBConnectionIdleTime: time.Minute,
		RedisHost: host, RedisPort: redisPort, RedisDB: redisDB, QueueRedisDB: queueDB,
		QueueConcurrency: 1, QueueShutdownTimeout: time.Second, RefreshTokenRetention: time.Hour,
		JWTSecret: strings.Repeat("s", 32), JWTIssuer: "integration-test", JWTAudience: "integration-test", JWTExpiration: time.Minute, RefreshExpiration: time.Hour,
		StorageRoot: t.TempDir(),
	}
	return cfg
}

func TestBootstrapConstructorsAndShutdown(t *testing.T) {
	cfg := testApplicationConfig(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	badDatabase := *cfg
	badDatabase.DBPort = 1
	if _, err := bootstrap.NewWorkerApplication(&badDatabase, logger); err == nil || !strings.Contains(err.Error(), "ping database") {
		t.Fatalf("NewWorkerApplication() with unavailable database error = %v", err)
	}
	badRedis := *cfg
	badRedis.RedisPort = 1
	if _, err := bootstrap.NewApplication(&badRedis); err == nil || !strings.Contains(err.Error(), "connect to Redis") {
		t.Fatalf("NewApplication() with unavailable Redis error = %v", err)
	}
	if _, err := bootstrap.NewWorkerApplication(&badRedis, logger); err == nil || !strings.Contains(err.Error(), "connect to queue Redis") {
		t.Fatalf("NewWorkerApplication() with unavailable Redis error = %v", err)
	}
	if _, err := bootstrap.NewSchedulerApplication(&badRedis, logger); err == nil || !strings.Contains(err.Error(), "connect to queue Redis") {
		t.Fatalf("NewSchedulerApplication() with unavailable Redis error = %v", err)
	}

	app, err := bootstrap.NewApplication(cfg)
	if err != nil {
		t.Fatalf("NewApplication(): %v", err)
	}
	if app.Router == nil || app.Server == nil || app.Queue == nil {
		t.Fatal("NewApplication() returned an incomplete application")
	}
	if err := app.Seed(context.Background()); err == nil || !strings.Contains(err.Error(), "ADMIN_PASSWORD") {
		t.Fatalf("Seed() error = %v, want missing admin password", err)
	}
	runResult := make(chan error, 1)
	go func() { runResult <- app.Run() }()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for {
		response, requestErr := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health/live", cfg.Port))
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("liveness status = %d, want 200", response.StatusCode)
			}
			break
		}
		select {
		case runErr := <-runResult:
			t.Fatalf("application stopped before serving HTTP: %v", runErr)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("application did not serve HTTP: %v", requestErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("Application.Shutdown(): %v", err)
	}
	if err := <-runResult; err != nil {
		t.Fatalf("Application.Run() after shutdown: %v", err)
	}

	workerApp, err := bootstrap.NewWorkerApplication(cfg, logger)
	if err != nil {
		t.Fatalf("NewWorkerApplication(): %v", err)
	}
	if workerApp.Worker == nil || workerApp.SQLDB == nil || workerApp.QueueRedis == nil {
		t.Fatal("NewWorkerApplication() returned an incomplete application")
	}
	if err := workerApp.Start(); err != nil {
		t.Fatalf("WorkerApplication.Start(): %v", err)
	}
	if err := workerApp.Shutdown(); err != nil {
		t.Fatalf("WorkerApplication.Shutdown(): %v", err)
	}

	schedulerApp, err := bootstrap.NewSchedulerApplication(cfg, logger)
	if err != nil {
		t.Fatalf("NewSchedulerApplication(): %v", err)
	}
	if schedulerApp.Scheduler == nil {
		t.Fatal("NewSchedulerApplication() returned an incomplete application")
	}
	if err := schedulerApp.Start(); err != nil {
		t.Fatalf("SchedulerApplication.Start(): %v", err)
	}
	if err := schedulerApp.Shutdown(); err != nil {
		t.Fatalf("SchedulerApplication.Shutdown(): %v", err)
	}
}
