//go:build integration

// Package testutil contains helpers used by integration tests. The build tag
// keeps the testcontainers dependency out of the default build graph.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewTestDB returns a pool pointed at the POSTGRES_DSN env var. We don't spin
// up containers in-process for the kit — CI provides Postgres via service
// containers and locally `make up` does the same.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		dsn = os.Getenv("POSTGRES_DSN")
	}
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN / POSTGRES_DSN not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

// ApplyMigrations runs all .up.sql files from dir against the pool. Crude on
// purpose — we only need this for tests.
func ApplyMigrations(t *testing.T, pool *pgxpool.Pool, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		if len(e.Name()) < 7 || e.Name()[len(e.Name())-7:] != ".up.sql" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if _, err := pool.Exec(context.Background(), string(raw)); err != nil {
			t.Fatalf("apply %s: %v", e.Name(), err)
		}
	}
}

// TruncateAll wipes every table in the public schema. Use between subtests.
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const q = `
		DO $$
		DECLARE r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname='public') LOOP
				EXECUTE format('TRUNCATE TABLE %I RESTART IDENTITY CASCADE', r.tablename);
			END LOOP;
		END $$;
	`
	if _, err := pool.Exec(context.Background(), q); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func RandomEmail() string {
	return fmt.Sprintf("user-%d@example.test", time.Now().UnixNano())
}
