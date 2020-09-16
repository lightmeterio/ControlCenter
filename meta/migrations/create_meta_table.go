package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util"
)

func init() {
	migrator.AddMigration("auth", "2_create_meta_table.go", UpMetaTable, DownMetaTable)
	migrator.AddMigration("master", "2_create_meta_table.go", UpMetaTable, DownMetaTable)
}

func UpMetaTable(tx *sql.Tx) error {

	sql := `create table if not exists users(
		email string,
		name string,
		password blob
	)`

	_, err := tx.Exec(sql)
	if err != nil {
		return util.WrapError(err)
	}
	return nil
}

func DownMetaTable(tx *sql.Tx) error {
	return nil
}
