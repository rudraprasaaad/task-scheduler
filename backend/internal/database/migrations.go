package database

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Migration struct {
	Version string
	Name    string
	SQL     string
}

type Migrator struct {
	db *DB
}

func NewMigrator(db *DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) CreateMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`

	_, err := m.db.Exec(query)

	if err != nil {
		return fmt.Errorf("failed to create migrations table:%w", err)
	}

	return nil
}

func (m *Migrator) GetAppliedMigrations() (map[string]bool, error) {
	applied := make(map[string]bool)

	query := "SELECT version FROM schema_migrations"
	rows, err := m.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations:%w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}

		applied[version] = true
	}

	return applied, nil
}

func (m *Migrator) ApplyMigration(migration Migration) error {
	tx, err := m.db.Begin()

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
	}

	insertQuery := `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`

	if _, err := tx.Exec(insertQuery, migration.Version, migration.Name); err != nil {
		return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
	}

	log.Printf("Applied migration: %s - %s", migration.Version, migration.Name)

	return nil
}

func (m *Migrator) RunMigrations(migrationsDir string) error {
	if err := m.CreateMigrationsTable(); err != nil {
		return err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	migrations, err := m.loadMigrations(migrationsDir)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if applied[migration.Version] {
			log.Printf("Migration %s already applied, skipped", migration.Version)
			continue
		}

		if err := m.ApplyMigration(migration); err != nil {
			return err
		}
	}

	log.Printf("All migrations completed successfully")
	return nil
}

func (m *Migrator) loadMigrations(dir string) ([]Migration, error) {
	var migrations []Migration

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := os.ReadFile(path)

		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", path, err)
		}

		fileName := d.Name()
		parts := strings.SplitN(fileName, "_", 2)

		if len(parts) < 2 {
			return fmt.Errorf("invalid migration filename format:%s", fileName)
		}

		version := parts[0]
		name := strings.TrimSuffix(parts[1], ".sql")

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
