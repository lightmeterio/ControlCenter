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
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/migrationutil"
)

func init() {
	migrator.AddMigration("master", "4_rename_postfix_ip_setting.go", upFixNames, downFixNames)
}

func updateContent(tx *sql.Tx, fixup func(string) string) (err error) {
	var value string

	err = tx.QueryRow(`select value from meta where key = 'global'`).Scan(&value)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	var v interface{}
	err = json.Unmarshal([]byte(value), &v)

	if err != nil {
		return errorutil.Wrap(err)
	}

	fixed, err := migrationutil.FixKeyNames(v, fixup)

	if err != nil {
		return errorutil.Wrap(err)
	}

	fixedValue, err := json.Marshal(fixed)
	if err != nil {
		return errorutil.Wrap(err)
	}

	_, err = tx.Exec(`update meta set value = ? where key = 'global'`, string(fixedValue))
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func upFixNames(tx *sql.Tx) error {
	return updateContent(tx, func(s string) string {
		if s == "local_ip" {
			return "postfix_public_ip"
		}

		return s
	})
}

func downFixNames(tx *sql.Tx) error {
	return updateContent(tx, func(s string) string {
		if s == "postfix_public_ip" {
			return "local_ip"
		}

		return s
	})
}
