// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("logs", "5_case_insensitive_domains_index.go", upCreateCaseInsensitiveDomainsIndex, downCreateCaseInsensitiveDomainsIndex)
}

func upCreateCaseInsensitiveDomainsIndex(tx *sql.Tx) error {
	sql := `CREATE INDEX domain_nocase_index ON remote_domains(domain COLLATE NOCASE)`

	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downCreateCaseInsensitiveDomainsIndex(tx *sql.Tx) error {
	return nil
}
