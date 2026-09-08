//go:build integration

package testdb

import (
	"context"
	"database/sql"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

// Tests create only random schemas on a loopback database explicitly supplied
// through DEUTERIUM_TEST_DSN. The production DSN is never read here.
func New(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("DEUTERIUM_TEST_DSN")
	if dsn == "" {
		t.Fatal("DEUTERIUM_TEST_DSN is required for integration tests")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid test DSN")
	}
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || cfg.DBName != "" || cfg.Net != "tcp" {
		t.Fatal("test database must be loopback TCP with no selected schema")
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	name := store.ID("deuterium_test_")
	if _, err = db.ExecContext(ctx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer db.Close()
		if !strings.HasPrefix(name, "deuterium_test_") {
			t.Fatal("unsafe test schema cleanup")
		}
		if _, err := db.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = name
	s, err := store.Open(ctx, cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.DB.Close() })
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return s
}
