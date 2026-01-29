package migrate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Migrator struct {
	db             *sql.DB
	migrationsPath string
}

type migration struct {
	version int
	name    string
	upSQL   string
	downSQL string
}

func NewMigrator(db *sql.DB, migrationsPath string) *Migrator {
	return &Migrator{
		db:             db,
		migrationsPath: migrationsPath,
	}
}

func (m *Migrator) ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) getAppliedVersions() (map[int]bool, error) {
	if err := m.ensureMigrationsTable(); err != nil {
		return nil, err
	}

	rows, err := m.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, nil
}

func parseVersion(versionStr string) (int, error) {
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return 0, fmt.Errorf("invalid version number: %s", versionStr)
	}
	return version, nil
}

func (m *Migrator) loadMigrations() ([]migration, error) {
	files, err := os.ReadDir(m.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrationsMap := make(map[int]*migration)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()

		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		var version int
		var direction, migrationName string

		if strings.HasSuffix(name, ".up.sql") {
			parts := strings.Split(name, "_")
			if len(parts) < 2 {
				fmt.Printf("Skipping invalid migration file: %s (expected format: NNN_name.up.sql)\n", name)
				continue
			}

			version, err = parseVersion(parts[0])
			if err != nil {
				fmt.Printf("Skipping file with invalid version: %s (%v)\n", name, err)
				continue
			}

			direction = "up"
			migrationName = strings.TrimSuffix(name, ".up.sql")
			migrationName = strings.TrimPrefix(migrationName, parts[0]+"_")
		} else if strings.HasSuffix(name, ".down.sql") {
			parts := strings.Split(name, "_")
			if len(parts) < 2 {
				fmt.Printf("Skipping invalid migration file: %s (expected format: NNN_name.down.sql)\n", name)
				continue
			}

			version, err = parseVersion(parts[0])
			if err != nil {
				fmt.Printf("Skipping file with invalid version: %s (%v)\n", name, err)
				continue
			}

			direction = "down"
			migrationName = strings.TrimSuffix(name, ".down.sql")
			migrationName = strings.TrimPrefix(migrationName, parts[0]+"_")
		} else {
			continue
		}

		content, err := os.ReadFile(filepath.Join(m.migrationsPath, name))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", name, err)
		}

		if migrationsMap[version] == nil {
			migrationsMap[version] = &migration{
				version: version,
				name:    migrationName,
			}
		}

		if direction == "up" {
			migrationsMap[version].upSQL = string(content)
			fmt.Printf("Loaded migration %d (%s) up: %d bytes\n", version, migrationName, len(content))
		} else {
			migrationsMap[version].downSQL = string(content)
			fmt.Printf("Loaded migration %d (%s) down: %d bytes\n", version, migrationName, len(content))
		}
	}

	migrations := make([]migration, 0, len(migrationsMap))
	for _, mig := range migrationsMap {
		migrations = append(migrations, *mig)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func (m *Migrator) Up() error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return err
	}

	for _, mig := range migrations {
		if applied[mig.version] {
			fmt.Printf("Migration %d (%s) already applied, skipping\n", mig.version, mig.name)
			continue
		}

		if err := m.applyMigration(mig, "up"); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) UpSteps(steps int) error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return err
	}

	count := 0
	for _, mig := range migrations {
		if applied[mig.version] {
			continue
		}

		if err := m.applyMigration(mig, "up"); err != nil {
			return err
		}

		count++
		if count >= steps {
			break
		}
	}

	return nil
}

func (m *Migrator) Down() error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return err
	}

	for i := len(migrations) - 1; i >= 0; i-- {
		mig := migrations[i]
		if !applied[mig.version] {
			fmt.Printf("Migration %d (%s) not applied, skipping\n", mig.version, mig.name)
			continue
		}

		if err := m.applyMigration(mig, "down"); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) DownSteps(steps int) error {
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return err
	}

	count := 0
	for i := len(migrations) - 1; i >= 0; i-- {
		mig := migrations[i]
		if !applied[mig.version] {
			continue
		}

		if err := m.applyMigration(mig, "down"); err != nil {
			return err
		}

		count++
		if count >= steps {
			break
		}
	}

	return nil
}

func (m *Migrator) applyMigration(mig migration, direction string) error {
	var sql string
	if direction == "up" {
		sql = mig.upSQL
	} else {
		sql = mig.downSQL
	}

	if sql == "" {
		return fmt.Errorf("no %s SQL found for migration %d (%s)", direction, mig.version, mig.name)
	}

	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	fmt.Printf("Running migration %d (%s) %s...\n", mig.version, mig.name, direction)

	if _, err := tx.Exec(sql); err != nil {
		return fmt.Errorf("failed to execute migration %d (%s) %s: %w", mig.version, mig.name, direction, err)
	}

	if direction == "up" {
		_, err = tx.Exec("INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", mig.version, mig.name)
	} else {
		_, err = tx.Exec("DELETE FROM schema_migrations WHERE version = $1", mig.version)
	}

	if err != nil {
		return fmt.Errorf("failed to update schema_migrations: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("Migration %d (%s) %s completed successfully!\n", mig.version, mig.name, direction)
	return nil
}
