package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("auth", "1_auth_migration_create_auth_table.go", UpUsersTable, DownUsersTable)
}

func UpUsersTable(tx *sql.Tx) error {

	sql := `create table if not exists users(
		email string,
		name string,
		password blob
	)`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}
	return nil
}

func DownUsersTable(tx *sql.Tx) error {
	return nil
}
