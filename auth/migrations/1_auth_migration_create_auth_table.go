package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
)

func init() {
	migrator.AddMigration("auth", "1_auth_migration_create_auth_table.go", Up, Down)
}

func Up(tx *sql.Tx) error {

	sql := `create table if not exists users(
		email string,
		name string,
		password blob
	)`

	_, err := tx.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func Down(tx *sql.Tx) error {

	sql := `DROP TABLE users;`

	_, err := tx.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}
