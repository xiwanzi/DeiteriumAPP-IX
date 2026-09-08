package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrConflict = errors.New("resource conflict")
var ErrUnauthorized = errors.New("unauthorized")
var ErrCredentialChanged = errors.New("credential changed during verification")
var ErrRateLimited = errors.New("rate limited")

type Store struct {
	DB             *sql.DB
	assetLifecycle sync.Mutex // The service already holds the database-wide runtime singleton lease.
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	if cfg.DBName == "" || cfg.MultiStatements || cfg.AllowAllFiles || cfg.AllowCleartextPasswords {
		return nil, errors.New("unsafe database configuration")
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.Timeout = 5 * time.Second
	cfg.ReadTimeout = 10 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["charset"] = "utf8mb4"
	cfg.Params["time_zone"] = "'+00:00'"
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, errors.New("database open failed")
	}
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.New("database connection failed")
	}
	return &Store{DB: db}, nil
}

func Digest(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

func (s *Store) Migrate(ctx context.Context) error {
	c, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	var lock int
	if err = c.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT(DATABASE(), ':deuterium-migrate'), 10)").Scan(&lock); err != nil || lock != 1 {
		return errors.New("migration lock unavailable")
	}
	defer c.ExecContext(context.Background(), "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':deuterium-migrate'))")
	_, err = c.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations_next (version VARCHAR(80) PRIMARY KEY, checksum CHAR(64) NOT NULL, applied_at DATETIME(6) NOT NULL) ENGINE=InnoDB`)
	if err != nil {
		return err
	}
	entries, _ := migrations.ReadDir("migrations")
	for _, entry := range entries {
		body, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		checksum := Digest(body)
		var existing string
		err = c.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations_next WHERE version=?", entry.Name()).Scan(&existing)
		if err == nil {
			if existing != checksum {
				return fmt.Errorf("migration checksum changed: %s", entry.Name())
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// Migrations contain idempotent DDL/index setup, no procedures or semicolons in strings.
		// MySQL DDL is not transactional: an interrupted migration is resumed safely.
		for _, statement := range strings.Split(string(body), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err = c.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migration %s failed: %w", entry.Name(), err)
			}
		}
		if _, err = c.ExecContext(ctx, "INSERT INTO schema_migrations_next VALUES (?,?,UTC_TIMESTAMP(6))", entry.Name(), checksum); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Ready(ctx context.Context) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		body, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return err
		}
		var checksum string
		if err = s.DB.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations_next WHERE version=?", entry.Name()).Scan(&checksum); err != nil {
			return err
		}
		if checksum != Digest(body) {
			return errors.New("schema checksum mismatch")
		}
	}
	return nil
}

func duplicate(err error) bool {
	var e *mysql.MySQLError
	return errors.As(err, &e) && e.Number == 1062
}

// Sweep only bounded indexed batches. Call outside request hot paths.
func (s *Store) Sweep(ctx context.Context) error {
	for _, q := range []string{
		"DELETE FROM auth_budgets WHERE expires_at < UTC_TIMESTAMP(6) LIMIT 1000",
		"DELETE FROM identity_sessions WHERE expires_at < UTC_TIMESTAMP(6) LIMIT 1000",
		"DELETE FROM core_chat_deliveries WHERE expires_at < DATE_SUB(UTC_TIMESTAMP(6), INTERVAL 1 DAY) LIMIT 1000",
	} {
		if _, err := s.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
