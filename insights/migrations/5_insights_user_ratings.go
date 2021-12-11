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
	migrator.AddMigration("insights", "5_insights_user_ratings.go", upRatings, downRatings)
}

func upRatings(tx *sql.Tx) error {
	sql := `
		create table insights_user_ratings(
			insight_type integer not null,
			rating integer not null,
			timestamp integer not null
		);

		create index insights_user_ratings_insight_type on insights_user_ratings(insight_type); 
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downRatings(tx *sql.Tx) error {
	_, err := tx.Exec(`drop table insights_user_ratings`)
	return err
}
