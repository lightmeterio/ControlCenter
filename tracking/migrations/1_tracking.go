package migrations

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

func init() {
	migrator.AddMigration("logtracker", "1_tracking.go", up, down)
}

func up(tx *sql.Tx) error {
	// TODO: investigate, via profiling, which fields deserve to have indexes apart from the obvious ones.
	sql := `
create table queues (
	connection_id integer not null,
	messageid_id integer,
	queue text not null
);

create index queue_text_index on queues(queue);

create table results (
	queue_id integer not null
);

create table result_data (
	result_id integer not null,
	key integer not null,
	value blob not null
);

create index result_data_result_id_index on result_data(result_id);

create table messageids (
	value text not null
);

create index messageids_text on messageids(value);

create table queue_parenting (
	orig_queue_id integer not null,
	new_queue_id integer not null,
	parenting_type integer not null
);

create table queue_data (
	queue_id integer not null,
	key integer not null,
	value blob not null
);

create index queue_data_result_id_index on queue_data(queue_id);

create table connections (
	pid_id integer not null
);

create index connections_pid_id_index on connections(pid_id);

create table connection_data (
	connection_id integer not null,
	key integer not null,
	value blob not null
);

create index connection_data_connection_id_index on connection_data(connection_id);

create table pids (
	pid integer not null,
	host text not null
);

create index pids_id_index on pids(host, pid);

-- TODO: move this table to a different file, or better, to memory!
create table notification_queues (
	result_id integer not null,
	line integer not null,
	filename text
);
`
	if _, err := tx.Exec(sql); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func down(tx *sql.Tx) error {
	return nil
}
