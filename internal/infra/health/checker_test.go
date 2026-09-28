package health

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

type pingConnector struct{ err error }

func (c pingConnector) Connect(context.Context) (driver.Conn, error) {
	return pingConnection{err: c.err}, nil
}
func (c pingConnector) Driver() driver.Driver { return pingDriver{err: c.err} }

type pingDriver struct{ err error }

func (d pingDriver) Open(string) (driver.Conn, error) { return pingConnection{err: d.err}, nil }

type pingConnection struct{ err error }

func (c pingConnection) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c pingConnection) Close() error                        { return nil }
func (c pingConnection) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (c pingConnection) Ping(context.Context) error          { return c.err }

func TestReadinessStopsWhenDatabasePingFails(t *testing.T) {
	dbError := errors.New("database unavailable")
	db := sql.OpenDB(pingConnector{err: dbError})
	t.Cleanup(func() { _ = db.Close() })
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	t.Cleanup(func() { _ = redisClient.Close() })

	err := NewChecker(db, redisClient).Readiness(context.Background())
	if !errors.Is(err, dbError) || !strings.Contains(err.Error(), "database is unavailable") {
		t.Fatalf("Readiness() error = %v", err)
	}
}

func TestReadinessReportsRedisPingFailureAfterDatabasePasses(t *testing.T) {
	db := sql.OpenDB(pingConnector{})
	t.Cleanup(func() { _ = db.Close() })
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0})
	if err := redisClient.Close(); err != nil {
		t.Fatal(err)
	}
	err := NewChecker(db, redisClient).Readiness(context.Background())
	if err == nil || !strings.Contains(err.Error(), "redis is unavailable") {
		t.Fatalf("Readiness() error = %v, want Redis failure", err)
	}
}
