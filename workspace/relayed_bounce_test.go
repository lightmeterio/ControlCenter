// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"os"
	"testing"
	"time"
)

func TestRelayedBounce(t *testing.T) {
	var year = 2020
	var (
		correctInterval = timeutil.TimeInterval{
			time.Date(year, time.January, 0, 0, 0, 0, 0, time.Local),
			time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local),
		}
	)

	Convey("Relayed bounce should update delivery status", t, func() {
		f, err := os.Open("../test_files/postfix_logs/individual_files/31_relayed_bounce.log")
		So(err, ShouldBeNil)

		// NOTE: detective sleep to leave time for the delivery to be created and updated
		d, clear := buildDetectiveFromReader(t, f, year)
		defer clear()

		messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, "", 1, limit)
		So(err, ShouldBeNil)

		So(messages.TotalResults, ShouldEqual, 2)

		msgDeliveries := messages.Messages[0].Entries
		So(len(msgDeliveries), ShouldEqual, 1)

		msgDelivery := msgDeliveries[0]
		So(msgDelivery.Dsn, ShouldEqual, "5.3.0")
		So(msgDelivery.Status, ShouldEqual, parser.BouncedStatus)
		So(len(msgDelivery.RawLogMsgs), ShouldEqual, 2)

		So(msgDelivery.RawLogMsgs[0], ShouldEqual, `Apr 27 12:46:31 lightmetermail postfix/smtp[12770]: 202A613D2BC: to=<h-4258189ab6952@h-9ce99f28.com>, relay=h-ff797cc8150da7ad8091.h-3d7c704422b72[135.237.217.50]:587, delay=2.3, delays=0.72/0.06/0.84/0.64, dsn=2.0.0, status=sent (250 Ok 010001806b0f0f99-41060108-df04-45ed-8b3c-e2f5d9275ec9-000000)`)

		// and we also get the lightmeter/relayed-bounce log line
		So(msgDelivery.RawLogMsgs[1], ShouldEqual, `Apr 27 12:46:34 lightmetermail lightmeter/relayed-bounce[62]: AC80013D2BC: Bounce: code="5.3.0", sender=<h-77ff@h-e0a7c0355e2339add.com>, recipient=<h-4258189ab6952@h-9ce99f28.com>, mta="h-676c0ed5fb08ee445040bc7ff918", message="554 rejected due to spam URL in content"`)
	})
}
