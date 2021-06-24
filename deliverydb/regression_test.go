// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

func TestRegresssion(t *testing.T) {
	Convey("Receive multiple delivery attempts from pre-filled results. Issue #516", t, func() {
		dir, clearDir := testutil.TempDir(t)
		defer clearDir()

		buildWs := func() (*DB, func() error, func(), tracking.ResultPublisher, dashboard.Dashboard, func()) {
			return buildWsFromDirectory(dir)
		}

		results, err := tracking.ParseResults([]string{
			`{
  "client_hostname": {
    "type": "text",
    "value": "h-083963ec00313e9"
  },
  "client_ip": {
    "type": "blob",
    "value": "AAAAAAAAAAAAAP//kzVLWg=="
  },
  "conn_ts_begin": {
    "type": "int64",
    "value": 1624165327
  },
  "conn_ts_end": {
    "type": "int64",
    "value": 1624165327
  },
  "connection_filename": {
    "type": "text",
    "value": "unknown"
  },
  "connection_line": {
    "type": "int64",
    "value": 1
  },
  "delay": {
    "type": "float64",
    "value": 0.10000000149011612
  },
  "delay_cleanup": {
    "type": "float64",
    "value": 0
  },
  "delay_qmgr": {
    "type": "float64",
    "value": 0
  },
  "delay_smtp": {
    "type": "float64",
    "value": 0.009999999776482582
  },
  "delay_smtpd": {
    "type": "float64",
    "value": 0.09000000357627869
  },
  "delivery_filename": {
    "type": "text",
    "value": "unknown"
  },
  "delivery_line": {
    "type": "int64",
    "value": 11
  },
  "delivery_queue": {
    "type": "text",
    "value": "95154657C"
  },
  "delivery_server": {
    "type": "text",
    "value": "ns4"
  },
  "delivery_ts": {
    "type": "int64",
    "value": 1624165327
  },
  "dsn": {
    "type": "text",
    "value": "2.0.0"
  },
  "message_direction": {
    "type": "int64",
    "value": 0
  },
  "message_id": {
    "type": "text",
    "value": "h-ec262eb25918e7678e9e8737f7b@h-e7d9fe256179482d76de1b3e83c.com"
  },
  "messageid_filename": {
    "type": "text",
    "value": "unknown"
  },
  "messageid_is_corrupted": {
    "type": "int64",
    "value": 0
  },
  "messageid_line": {
    "type": "int64",
    "value": 4
  },
  "nrcpt": {
    "type": "int64",
    "value": 1
  },
  "orig_recipient_domain_part": {
    "type": "text",
    "value": "h-20b651e8120a33ec11.com"
  },
  "orig_recipient_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "orig_size": {
    "type": "int64",
    "value": 1605
  },
  "processed_size": {
    "type": "int64",
    "value": 1605
  },
  "queue_commit_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_commit_line": {
    "type": "int64",
    "value": 12
  },
  "queue_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_line": {
    "type": "int64",
    "value": 3
  },
  "queue_ts_begin": {
    "type": "int64",
    "value": 1624165327
  },
  "queue_ts_end": {
    "type": "int64",
    "value": 1624165327
  },
  "recipient_domain_part": {
    "type": "text",
    "value": "h-ea3f4afa.com"
  },
  "recipient_local_part": {
    "type": "text",
    "value": "h-493fac8f3"
  },
  "sender_domain_part": {
    "type": "text",
    "value": "h-b7bed8eb24c5049d9.com"
  },
  "sender_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "status": {
    "type": "int64",
    "value": 0
  }
}`,
			`{
  "client_hostname": {
    "type": "text",
    "value": "h-df09b377defe41d"
  },
  "client_ip": {
    "type": "blob",
    "value": "AAAAAAAAAAAAAP//kzVLWg=="
  },
  "conn_ts_begin": {
    "type": "int64",
    "value": 1624165354
  },
  "conn_ts_end": {
    "type": "int64",
    "value": 1624165355
  },
  "connection_filename": {
    "type": "text",
    "value": "unknown"
  },
  "connection_line": {
    "type": "int64",
    "value": 15
  },
  "delay": {
    "type": "float64",
    "value": 0.25999999046325684
  },
  "delay_cleanup": {
    "type": "float64",
    "value": 0.009999999776482582
  },
  "delay_qmgr": {
    "type": "float64",
    "value": 0.12999999523162842
  },
  "delay_smtp": {
    "type": "float64",
    "value": 0.019999999552965164
  },
  "delay_smtpd": {
    "type": "float64",
    "value": 0.09000000357627869
  },
  "delivery_filename": {
    "type": "text",
    "value": "unknown"
  },
  "delivery_line": {
    "type": "int64",
    "value": 26
  },
  "delivery_queue": {
    "type": "text",
    "value": "E389F657C"
  },
  "delivery_server": {
    "type": "text",
    "value": "ns4"
  },
  "delivery_ts": {
    "type": "int64",
    "value": 1624165355
  },
  "dsn": {
    "type": "text",
    "value": "2.0.0"
  },
  "message_direction": {
    "type": "int64",
    "value": 0
  },
  "message_id": {
    "type": "text",
    "value": "h-2168aefc624da54daa88a302b8@h-bf5fc94ab1f34a49c62aff1c094.com"
  },
  "messageid_filename": {
    "type": "text",
    "value": "unknown"
  },
  "messageid_is_corrupted": {
    "type": "int64",
    "value": 0
  },
  "messageid_line": {
    "type": "int64",
    "value": 18
  },
  "nrcpt": {
    "type": "int64",
    "value": 1
  },
  "orig_recipient_domain_part": {
    "type": "text",
    "value": "h-20b651e8120a33ec11.com"
  },
  "orig_recipient_local_part": {
    "type": "text",
    "value": "h-94723d9c85e1"
  },
  "orig_size": {
    "type": "int64",
    "value": 1630
  },
  "processed_size": {
    "type": "int64",
    "value": 1630
  },
  "queue_commit_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_commit_line": {
    "type": "int64",
    "value": 27
  },
  "queue_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_line": {
    "type": "int64",
    "value": 17
  },
  "queue_ts_begin": {
    "type": "int64",
    "value": 1624165354
  },
  "queue_ts_end": {
    "type": "int64",
    "value": 1624165355
  },
  "recipient_domain_part": {
    "type": "text",
    "value": "h-a85427309c70.com"
  },
  "recipient_local_part": {
    "type": "text",
    "value": "h-94723d9c85e1"
  },
  "relay_ip": {
    "type": "blob",
    "value": "AAAAAAAAAAAAAP//xwP+mw=="
  },
  "relay_name": {
    "type": "text",
    "value": "h-a85427309c70"
  },
  "relay_port": {
    "type": "int64",
    "value": 25
  },
  "sender_domain_part": {
    "type": "text",
    "value": "h-ca65051b52b067e2c8975.com"
  },
  "sender_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "status": {
    "type": "int64",
    "value": 0
  }
}`,
			`{
  "delay": {
    "type": "float64",
    "value": 0.25999999046325684
  },
  "delay_cleanup": {
    "type": "float64",
    "value": 0
  },
  "delay_qmgr": {
    "type": "float64",
    "value": 0.2199999988079071
  },
  "delay_smtp": {
    "type": "float64",
    "value": 0.029999999329447746
  },
  "delay_smtpd": {
    "type": "float64",
    "value": 0.009999999776482582
  },
  "delivery_filename": {
    "type": "text",
    "value": "unknown"
  },
  "delivery_line": {
    "type": "int64",
    "value": 38
  },
  "delivery_queue": {
    "type": "text",
    "value": "848F1657D"
  },
  "delivery_server": {
    "type": "text",
    "value": "ns4"
  },
  "delivery_ts": {
    "type": "int64",
    "value": 1624165375
  },
  "dsn": {
    "type": "text",
    "value": "2.0.0"
  },
  "message_direction": {
    "type": "int64",
    "value": 0
  },
  "message_id": {
    "type": "text",
    "value": "h-9723265c531c21306faa0968@h-986d88a6b79105.com"
  },
  "messageid_filename": {
    "type": "text",
    "value": "unknown"
  },
  "messageid_is_corrupted": {
    "type": "int64",
    "value": 0
  },
  "messageid_line": {
    "type": "int64",
    "value": 33
  },
  "nrcpt": {
    "type": "int64",
    "value": 1
  },
  "orig_recipient_domain_part": {
    "type": "text",
    "value": ""
  },
  "orig_recipient_local_part": {
    "type": "text",
    "value": ""
  },
  "orig_size": {
    "type": "int64",
    "value": 1221
  },
  "pickup_sender": {
    "type": "text",
    "value": "h-195704c@h-20b651e8120a33ec11.com"
  },
  "pickup_uid": {
    "type": "int64",
    "value": 2002
  },
  "processed_size": {
    "type": "int64",
    "value": 1221
  },
  "queue_commit_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_commit_line": {
    "type": "int64",
    "value": 39
  },
  "queue_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_line": {
    "type": "int64",
    "value": 32
  },
  "queue_ts_begin": {
    "type": "int64",
    "value": 1624165375
  },
  "queue_ts_end": {
    "type": "int64",
    "value": 1624165375
  },
  "recipient_domain_part": {
    "type": "text",
    "value": "h-b7bed8eb24c5049d9.com"
  },
  "recipient_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "relay_ip": {
    "type": "blob",
    "value": "AAAAAAAAAAAAAP//kzVLJg=="
  },
  "relay_name": {
    "type": "text",
    "value": "h-7a12f4efdf064cf"
  },
  "relay_port": {
    "type": "int64",
    "value": 25
  },
  "sender_domain_part": {
    "type": "text",
    "value": "h-20b651e8120a33ec11.com"
  },
  "sender_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "status": {
    "type": "int64",
    "value": 0
  }
}`,
			`{
  "delay": {
    "type": "float64",
    "value": 0.23000000417232513
  },
  "delay_cleanup": {
    "type": "float64",
    "value": 0
  },
  "delay_qmgr": {
    "type": "float64",
    "value": 0.1899999976158142
  },
  "delay_smtp": {
    "type": "float64",
    "value": 0.029999999329447746
  },
  "delay_smtpd": {
    "type": "float64",
    "value": 0.009999999776482582
  },
  "delivery_filename": {
    "type": "text",
    "value": "unknown"
  },
  "delivery_line": {
    "type": "int64",
    "value": 46
  },
  "delivery_queue": {
    "type": "text",
    "value": "96C82657D"
  },
  "delivery_server": {
    "type": "text",
    "value": "ns4"
  },
  "delivery_ts": {
    "type": "int64",
    "value": 1624165441
  },
  "dsn": {
    "type": "text",
    "value": "2.0.0"
  },
  "message_direction": {
    "type": "int64",
    "value": 0
  },
  "message_id": {
    "type": "text",
    "value": "h-372f68c1549c5fea48442c08@h-986d88a6b79105.com"
  },
  "messageid_filename": {
    "type": "text",
    "value": "unknown"
  },
  "messageid_is_corrupted": {
    "type": "int64",
    "value": 0
  },
  "messageid_line": {
    "type": "int64",
    "value": 41
  },
  "nrcpt": {
    "type": "int64",
    "value": 1
  },
  "orig_recipient_domain_part": {
    "type": "text",
    "value": ""
  },
  "orig_recipient_local_part": {
    "type": "text",
    "value": ""
  },
  "orig_size": {
    "type": "int64",
    "value": 1239
  },
  "pickup_sender": {
    "type": "text",
    "value": "h-195704c@h-20b651e8120a33ec11.com"
  },
  "pickup_uid": {
    "type": "int64",
    "value": 2002
  },
  "processed_size": {
    "type": "int64",
    "value": 1239
  },
  "queue_commit_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_commit_line": {
    "type": "int64",
    "value": 47
  },
  "queue_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_line": {
    "type": "int64",
    "value": 40
  },
  "queue_ts_begin": {
    "type": "int64",
    "value": 1624165441
  },
  "queue_ts_end": {
    "type": "int64",
    "value": 1624165441
  },
  "recipient_domain_part": {
    "type": "text",
    "value": "h-b7bed8eb24c5049d9.com"
  },
  "recipient_local_part": {
    "type": "text",
    "value": "h-ed3f76aa9e787478a8"
  },
  "relay_ip": {
    "type": "blob",
    "value": "AAAAAAAAAAAAAP//kzVLJg=="
  },
  "relay_name": {
    "type": "text",
    "value": "h-7a12f4efdf064cf"
  },
  "relay_port": {
    "type": "int64",
    "value": 25
  },
  "sender_domain_part": {
    "type": "text",
    "value": "h-20b651e8120a33ec11.com"
  },
  "sender_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "status": {
    "type": "int64",
    "value": 0
  }
}`,
			`{
  "client_hostname": {
    "type": "text",
    "value": "h-9421d836902d257"
  },
  "client_ip": {
    "type": "blob",
    "value": "AAAAAAAAAAAAAP//kzVLUw=="
  },
  "conn_ts_begin": {
    "type": "int64",
    "value": 1624165447
  },
  "conn_ts_end": {
    "type": "int64",
    "value": 1624165447
  },
  "connection_filename": {
    "type": "text",
    "value": "unknown"
  },
  "connection_line": {
    "type": "int64",
    "value": 51
  },
  "delay": {
    "type": "float64",
    "value": 0.10999999940395355
  },
  "delay_cleanup": {
    "type": "float64",
    "value": 0.009999999776482582
  },
  "delay_qmgr": {
    "type": "float64",
    "value": 0
  },
  "delay_smtp": {
    "type": "float64",
    "value": 0.009999999776482582
  },
  "delay_smtpd": {
    "type": "float64",
    "value": 0.10000000149011612
  },
  "delivery_filename": {
    "type": "text",
    "value": "unknown"
  },
  "delivery_line": {
    "type": "int64",
    "value": 61
  },
  "delivery_queue": {
    "type": "text",
    "value": "D390B657C"
  },
  "delivery_server": {
    "type": "text",
    "value": "ns4"
  },
  "delivery_ts": {
    "type": "int64",
    "value": 1624165447
  },
  "dsn": {
    "type": "text",
    "value": "2.0.0"
  },
  "message_direction": {
    "type": "int64",
    "value": 0
  },
  "message_id": {
    "type": "text",
    "value": "h-dfd067542de35f4b23673e0b3b3@h-e7d9fe256179482d76de1b3e83c.com"
  },
  "messageid_filename": {
    "type": "text",
    "value": "unknown"
  },
  "messageid_is_corrupted": {
    "type": "int64",
    "value": 0
  },
  "messageid_line": {
    "type": "int64",
    "value": 54
  },
  "nrcpt": {
    "type": "int64",
    "value": 1
  },
  "orig_recipient_domain_part": {
    "type": "text",
    "value": "h-20b651e8120a33ec11.com"
  },
  "orig_recipient_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "orig_size": {
    "type": "int64",
    "value": 1605
  },
  "processed_size": {
    "type": "int64",
    "value": 1605
  },
  "queue_commit_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_commit_line": {
    "type": "int64",
    "value": 62
  },
  "queue_filename": {
    "type": "text",
    "value": "unknown"
  },
  "queue_line": {
    "type": "int64",
    "value": 53
  },
  "queue_ts_begin": {
    "type": "int64",
    "value": 1624165447
  },
  "queue_ts_end": {
    "type": "int64",
    "value": 1624165447
  },
  "recipient_domain_part": {
    "type": "text",
    "value": "h-ea3f4afa.com"
  },
  "recipient_local_part": {
    "type": "text",
    "value": "h-493fac8f3"
  },
  "sender_domain_part": {
    "type": "text",
    "value": "h-b7bed8eb24c5049d9.com"
  },
  "sender_local_part": {
    "type": "text",
    "value": "h-195704c"
  },
  "status": {
    "type": "int64",
    "value": 0
  }
}`,
		})

		So(err, ShouldBeNil)

		db, done, cancel, pub, dashboard, dtor := buildWs()
		defer dtor()

		for _, r := range results {
			pub.Publish(r)
		}

		cancel()
		So(done(), ShouldBeNil)

		interval := parseTimeInterval("0000-01-01", "4000-01-01")

		So(db.HasLogs(), ShouldBeTrue)
		So(countByStatus(dashboard, parser.BouncedStatus, interval), ShouldEqual, 0)
		So(countByStatus(dashboard, parser.DeferredStatus, interval), ShouldEqual, 0)
		So(countByStatus(dashboard, parser.SentStatus, interval), ShouldEqual, 5)
	})
}
