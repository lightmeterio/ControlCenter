package migrations

import (
	"database/sql"
)

func UpMetaTable(tx *sql.Tx) error {

	sql := `create table if not exists meta(
		key string,
		value blob
	)`

	_, err := tx.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func DownMetaTable(tx *sql.Tx) error {

	sql := `DROP TABLE meta;`

	_, err := tx.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}
