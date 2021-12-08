// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
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
	migrator.AddMigration("insights", "1_insights.go", upInsights, downInsights)
}

func upInsights(tx *sql.Tx) error {
	sql := `create table insights(
			time integer not null,
			category integer not null,
			rating integer not null,
			content_type integer not null,
			content blob not null
		);

		create index insights_time_index on insights(time); 

		create index insights_category_index on insights(category, time);

		create index insights_rating_index on insights(rating, time);

		create index insights_content_type_index on insights(content_type, time);

		create table last_detector_execution(ts integer, kind text)
		`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downInsights(tx *sql.Tx) error {
	return nil
}
