package migrations

import (
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("deliverydb", "1_create_table_postfix_smtp_message_status.go", upCreateOldLogTables, downCreateOldLogTables)
}

func upCreateOldLogTables(tx *sql.Tx) error {
	sql := `
	create table if not exists postfix_smtp_message_status(
		read_ts_sec           integer,
		process_ip            blob,
		queue                 string,
		recipient_local_part  text,
		recipient_domain_part text,
		relay_name            text,
		relay_ip              blob,
		relay_port            uint16,
		delay                 double,
		delay_smtpd           double,
		delay_cleanup         double,
		delay_qmgr            double,
		delay_smtp            double,
		dsn                   text,
		status                integer
	);
		
	create index if not exists postfix_smtp_message_time_index on postfix_smtp_message_status (read_ts_sec);
`

	_, err := tx.Exec(sql)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func downCreateOldLogTables(tx *sql.Tx) error {
	return errors.New(`Cannot migrate down from the first database schema`)
}
