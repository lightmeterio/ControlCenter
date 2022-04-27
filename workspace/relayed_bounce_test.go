// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRelayedBounce(t *testing.T) {
	var year = time.Now().Year()
	var (
		correctInterval = timeutil.TimeInterval{
			time.Date(year, time.January, 0, 0, 0, 0, 0, time.Local),
			time.Date(year, time.December, 31, 0, 0, 0, 0, time.Local),
		}
	)

	Convey("Relayed bounce should update delivery status", t, func() {
		f, err := os.Open("../test_files/postfix_logs/individual_files/31_relayed_bounce.log")
		So(err, ShouldBeNil)

		text, err := io.ReadAll(f)
		So(err, ShouldBeNil)

		// Status will only be updated if delivery is less than 1h-old
		halfAnHourAgo := time.Now().UTC().Add(-30 * time.Minute)
		modifiedLogs := strings.ReplaceAll(string(text), "Apr 26 19:16", halfAnHourAgo.Format("Jan 02 15:04"))

		// NOTE: detective sleep to leave time for the delivery to be created and updated
		d, clear := buildDetectiveFromReader(t, strings.NewReader(modifiedLogs), year, 2*time.Second)
		defer clear()

		messages, err := d.CheckMessageDelivery(bg, "", "", correctInterval, -1, "", 1, limit)
		So(err, ShouldBeNil)

		So(messages.TotalResults, ShouldEqual, 1)

		msgDeliveries := messages.Messages[0].Entries
		So(len(msgDeliveries), ShouldEqual, 1)

		msgDelivery := msgDeliveries[0]
		So(msgDelivery.Dsn, ShouldEqual, "5.6.7")
		So(msgDelivery.Status, ShouldEqual, parser.BouncedStatus)
		So(len(msgDelivery.RawLogMsgs), ShouldEqual, 2)

		firstLineNoDate := `lightmetermail postfix/smtp[34591]: AEB0817CED6: to=<h-85aced2@h-04ec078cf6df16e03c.com>, relay=h-ff797cc8150da7ad8091.h-3d7c704422b72[83.244.15.1]:587, delay=2.1, delays=0.62/0.05/0.86/0.6, dsn=2.0.0, status=sent (250 Ok 010001805f291096-faccf9fa-b272-4ef7-bd4b-6a072596174a-000000)`

		// and we also get the lightmeter/relayed-bounce log line
		secondLineNoDate := `lightmetermail lightmeter/relayed-bounce[54]: Bounce: code="5.6.7", sender=<h-77ff@h-e0a7c0355e2339add.com>, recipient=<h-85aced2@h-04ec078cf6df16e03c.com>, mta="h-6668c859feb3a3fa468d05.h-9f8f068470e0a72b", message="550 5.6.7 DNS domain h-97a777c7701de3e5b29a1b917bf does not exist [Message=InfoDomainNonexistent] [LastAttemptedServerName=h-97a777c7701de3e5b29a1b917bf] [h-ea6044783cd18cd236eb644.h-ae356a81c0f1953ae829d6902d5]"`

		So(strings.HasSuffix(msgDelivery.RawLogMsgs[0], firstLineNoDate), ShouldBeTrue)
		So(strings.HasSuffix(msgDelivery.RawLogMsgs[1], secondLineNoDate), ShouldBeTrue)
	})
}
