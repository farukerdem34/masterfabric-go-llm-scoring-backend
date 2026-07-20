package postgres

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations applies all pending SQL migrations from the embedded migrations/ directory.
// Set force=true to drop schema_migrations and re-apply all (use when tables were lost).
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, force bool) error {
	if force {
		log.Warn("force migration: dropping schema_migrations table")
		if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations"); err != nil {
			return fmt.Errorf("drop schema_migrations: %w", err)
		}
	}

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	applied := make(map[string]bool)
	rows, err := pool.Query(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("query schema_migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = true
	}
	rows.Close()

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		if applied[name] {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		sql := stripGooseDirectives(string(content))

		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", name); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		log.Info("migration applied", "file", name)
	}

	log.Info("all migrations up to date", "total", len(files))
	return nil
}

func stripGooseDirectives(sql string) string {
	var upLines []string
	inUp := false
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "-- +goose Up" {
			inUp = true
			continue
		}
		if trimmed == "-- +goose Down" {
			inUp = false
			continue
		}
		if inUp {
			upLines = append(upLines, line)
		}
	}
	return strings.Join(upLines, "\n")
}
