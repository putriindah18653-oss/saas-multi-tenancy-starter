package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/putriindah18653-oss/saas-multi-tenancy-starter/backend-api/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/migrate up|down")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	// Ensure schema_migrations table exists for version tracking.
	_, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`)
	if err != nil {
		log.Fatalf("ensure schema_migrations table: %v", err)
	}

	dir := "migrations"
	suffix := "." + os.Args[1] + ".sql"
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), suffix) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if os.Args[1] == "down" {
		for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
			names[i], names[j] = names[j], names[i]
		}
	}
	for _, name := range names {
		version := strings.TrimSuffix(name, suffix)

		if os.Args[1] == "up" {
			var exists bool
			err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", version).Scan(&exists)
			if err != nil {
				log.Fatalf("check version %s: %v", version, err)
			}
			if exists {
				fmt.Println("skip", name, "(already applied)")
				continue
			}
		}

		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("apply", name)
		if _, err := conn.Exec(ctx, string(b)); err != nil {
			log.Fatalf("%s: %v", name, err)
		}

		if os.Args[1] == "up" {
			if _, err := conn.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", version); err != nil {
				log.Fatalf("record version %s: %v", version, err)
			}
		} else {
			if _, err := conn.Exec(ctx, "DELETE FROM schema_migrations WHERE version=$1", version); err != nil {
				log.Fatalf("remove version %s: %v", version, err)
			}
		}

		fmt.Println("done", name)
	}
}
