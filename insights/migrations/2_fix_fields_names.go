// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

/**
 * The high_status_rate and the mail_inactivity insights store json values with some field names
 * that don't follow out convention (snake_case), using CamelCase instead.
 * Such values are exposed in the HTTP API, so case convention is important.

 * This migration aims to fix it, by replacing such values direct in the database.
 */

import (
	"database/sql"
	"encoding/json"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/migrationutil"
)

func init() {
	migrator.AddMigration("insights", "2_fix_fields_names.go", upFixNames, downFixNames)
}

func updateContent(tx *sql.Tx, fixup func(string) string) (err error) {
	rows, err := tx.Query(`select rowid, time, content from insights`)
	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errorutil.Wrap(closeErr)
		}
	}()

	for rows.Next() {
		var (
			rowid   int64
			time    int64
			content string
		)

		err = rows.Scan(&rowid, &time, &content)
		if err != nil {
			return errorutil.Wrap(err)
		}

		var v interface{}
		err = json.Unmarshal([]byte(content), &v)

		if err != nil {
			return errorutil.Wrap(err)
		}

		fixedContent, err := migrationutil.FixKeyNames(v, fixup)
		if err != nil {
			return errorutil.Wrap(err)
		}

		encodedValue, err := json.Marshal(fixedContent)
		if err != nil {
			return errorutil.Wrap(err)
		}

		_, err = tx.Exec(`update insights set content = ? where rowid = ?`, encodedValue, rowid)
		if err != nil {
			return errorutil.Wrap(err)
		}
	}

	if err := rows.Err(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func buildFixUp(m map[string]string) func(string) string {
	return func(s string) string {
		if v, ok := m[s]; ok {
			return v
		}

		return s
	}
}

var camelToSnakeCase = map[string]string{
	"From":     "from",
	"To":       "to",
	"Value":    "value",
	"Interval": "interval",
}

func upFixNames(tx *sql.Tx) error {
	return updateContent(tx, buildFixUp(camelToSnakeCase))
}

// On downgrading versions, we want to update back only a few field names,
// leaving the others untouched
var snakeToCamelCase = map[string]string{
	"from":     "From",
	"to":       "To",
	"value":    "Value",
	"interval": "Interval",
}

func downFixNames(tx *sql.Tx) error {
	return updateContent(tx, buildFixUp(snakeToCamelCase))
}
