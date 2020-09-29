package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("auth", "3_auth_migration_clear_old_keys.go", upClearSessionKeys, downClearSessionKeys)
}

// We changed the whey the sesssion keys are stored, from plain to json,
// so we need to clean them. All users will be delogged, and new keys
// will automatically be generated
func upClearSessionKeys(tx *sql.Tx) error {
	_, err := tx.Exec(`delete from meta where key = ?`, "session_key")
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downClearSessionKeys(tx *sql.Tx) error {
	// No way to recover the old session keys, but that's fine,
	// as they'll be automatically regenerated
	return nil
}
