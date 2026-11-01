package database

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The Aurora fork's own schema changes run after Polyfin's migrations, in
// the order of their names, as aurora_migrations records them: kept apart,
// they never take the number of Polyfin's next migration, which the daily
// merge of Polyfin brings. Each applies to a database that may have it
// already, and runs again when it was edited.

//go:embed aurora/*.sql
var auroraFiles embed.FS

// auroraMigrations are the fork's schema changes, in order.
func auroraMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(auroraFiles, "aurora")
	if err != nil {
		return nil, err
	}
	var migrations []migration
	for _, entry := range entries {
		match := migrationName.FindStringSubmatch(entry.Name())
		if match == nil || entry.IsDir() {
			return nil, fmt.Errorf("Aurora migration %q is not named NNNN_name.sql", entry.Name())
		}
		body, err := fs.ReadFile(auroraFiles, "aurora/"+entry.Name())
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(body)
		migrations = append(migrations, migration{name: entry.Name(), sql: string(body), checksum: hex.EncodeToString(sum[:])})
	}
	slices.SortFunc(migrations, func(a, b migration) int { return strings.Compare(a.name, b.name) })
	return migrations, nil
}

// migrateAurora applies the fork's schema changes not applied yet, or
// edited since, each in its own transaction, under the migrations' lock.
func migrateAurora(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := auroraMigrations()
	if err != nil {
		return err
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLock); err != nil {
		return fmt.Errorf("lock the schema: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", migrationLock); unlockErr != nil {
			_ = conn.Conn().Close(context.WithoutCancel(ctx))
		}
	}()
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS aurora_migrations (
		name text PRIMARY KEY,
		checksum text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create aurora_migrations: %w", err)
	}
	rows, _ := conn.Query(ctx, "SELECT name, checksum FROM aurora_migrations")
	applied := map[string]string{}
	var name, checksum string
	if _, err := pgx.ForEachRow(rows, []any{&name, &checksum}, func() error {
		applied[name] = checksum
		return nil
	}); err != nil {
		return fmt.Errorf("read aurora_migrations: %w", err)
	}
	for _, m := range migrations {
		if applied[m.name] == m.checksum {
			continue
		}
		err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, m.sql); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO aurora_migrations (name, checksum) VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET checksum = excluded.checksum, applied_at = now()`, m.name, m.checksum)
			return err
		})
		if err != nil {
			return fmt.Errorf("apply Aurora migration %s: %w", m.name, err)
		}
	}
	return nil
}
