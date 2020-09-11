package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
)

func init() {
	migrator.AddMigration("insights", "1599813865_insights.go", UpInsights, DownInsights)
}

func UpInsights(tx *sql.Tx) error {

	sql := `create table if not exists insights(
			time integer not null,
			category integer not null,
			priority integer not null,
			content_type integer not null,
			content blob not null
		);

		create index if not exists insights_time_index on insights(time); 

		create index if not exists insights_category_index on insights(category, time);

		create index if not exists insights_priority_index on insights(priority, time);

		create index if not exists insights_content_type_index on insights(content_type, time);

		create table if not exists last_detector_execution(ts integer, kind text)
		`

	_, err := tx.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func DownInsights(tx *sql.Tx) error {

	sql := `
		drop index insights_time_index; 
		drop index insights_category_index;
        drop index insights_priority_index;
        drop index insights_content_type_index;
        drop table insights;
		drop table last_detector_execution;
        `

	_, err := tx.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}
