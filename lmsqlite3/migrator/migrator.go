package migrator

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/pressly/goose"
	"log"
	"math"
	"path/filepath"
	"time"

	"sort"
)

func Run(database *sql.DB, databaseName string) error {

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	var err error
	err = Status(database, databaseName)
	if err != nil {
		return fmt.Errorf("could not get database migration status: %w", err)
	}

	err = Up(database, databaseName)
	if err != nil {
		return fmt.Errorf("could not run goose up: %w", err)
	}

	return nil
}

var (
	registeredGoMigrations = map[string]map[int64]*goose.Migration{}
)

// MaxVersion is the maximum allowed version.
const MaxVersion int64 = math.MaxInt64
const minVersion = int64(0)

// Up migrates up to a specific version.
func Up(db *sql.DB, databaseName string) error {
	migrations, err := CollectMigrations(minVersion, MaxVersion, databaseName)
	if err != nil {
		return err
	}

	for {
		current, err := goose.GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == goose.ErrNoNextVersion {
				log.Printf("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}

		if err = next.Up(db); err != nil {
			return err
		}
	}
}

// CollectMigrations returns all the valid looking migration scripts in the
// migrations folder and go func registry, and key them by version.
func CollectMigrations(current, target int64, databaseName string) (goose.Migrations, error) {

	var migrations goose.Migrations

	// Go migrations registered via goose.AddMigration().
	for _, migration := range registeredGoMigrations[databaseName] {
		v, err := goose.NumericComponent(migration.Source)
		if err != nil {
			return nil, err
		}
		if versionFilter(v, current, target) {
			migrations = append(migrations, migration)
		}
	}

	migrations = sortAndConnectMigrations(migrations)

	return migrations, nil
}

func sortAndConnectMigrations(migrations goose.Migrations) goose.Migrations {
	sort.Sort(migrations)

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range migrations {
		prev := int64(-1)
		if i > 0 {
			prev = migrations[i-1].Version
			migrations[i-1].Next = m.Version
		}
		migrations[i].Previous = prev
	}

	return migrations
}

func versionFilter(v, current, target int64) bool {

	if target > current {
		return v > current && v <= target
	}

	if target < current {
		return v <= current && v > target
	}

	return false
}

// Status prints the status of all migrations.
func Status(db *sql.DB, databaseName string) error {
	// collect all migrations
	migrations, err := CollectMigrations(minVersion, MaxVersion, databaseName)
	if err != nil {
		return errors.Wrap(err, "failed to collect migrations")
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := goose.EnsureDBVersion(db); err != nil {
		return errors.Wrap(err, "failed to ensure DB version")
	}

	log.Print("\n")

	log.Println(fmt.Sprintf("    Database name               %v", databaseName))
	log.Println("    Applied At                  Migration")
	log.Println("    =======================================")
	for _, migration := range migrations {
		if err := printMigrationStatus(db, migration.Version, filepath.Base(migration.Source)); err != nil {
			return errors.Wrap(err, "failed to print status")
		}
	}
	return nil
}

func printMigrationStatus(db *sql.DB, version int64, script string) error {
	/* #nosec */
	q := fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=%d ORDER BY tstamp DESC LIMIT 1", goose.TableName(), version)

	var row goose.MigrationRecord
	err := db.QueryRow(q).Scan(&row.TStamp, &row.IsApplied)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "failed to query the latest migration")
	}

	var appliedAt string
	if row.IsApplied {
		appliedAt = row.TStamp.Format(time.ANSIC)
	} else {
		appliedAt = "Pending"
	}

	log.Printf("    %-24s -- %v\n", appliedAt, script)
	return nil
}

// AddMigration : Add a migration.
func AddMigration(databaseName string, filename string, up func(*sql.Tx) error, down func(*sql.Tx) error) {

	v, _ := goose.NumericComponent(filename)
	migration := &goose.Migration{Version: v, Next: -1, Previous: -1, Registered: true, UpFn: up, DownFn: down, Source: filename}

	if _, ok := registeredGoMigrations[databaseName]; !ok {
		registeredGoMigrations[databaseName] = map[int64]*goose.Migration{}
	}

	registeredGoMigrations[databaseName][v] = migration
}
