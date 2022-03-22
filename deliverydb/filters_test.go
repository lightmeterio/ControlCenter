// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"testing"
)

func TestFilters(t *testing.T) {
	// FIXME: meh! those tests are very ugly and covering very few cases!!!
	Convey("Test Filters", t, func() {
		filters, err := BuildFilters(FiltersDescription{
			Rule1: FilterDescription{AcceptSender: "accept_sender@example1.com"},
			Rule2: FilterDescription{RejectRecipient: "reject_recipient1@example2.com"},
			Rule3: FilterDescription{RejectRecipient: "reject_recipient2@example3.com"},
		})

		So(err, ShouldBeNil)

		So(filters.Reject(tracking.MappedResult{
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient1"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example2.com"),
		}.Result()), ShouldBeTrue)

		So(filters.Reject(tracking.MappedResult{
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example3.com"),
		}.Result()), ShouldBeTrue)

		So(filters.Reject(tracking.MappedResult{
			tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
			tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
			tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("another_recipient2"),
			tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example3.com"),
		}.Result()), ShouldBeFalse)
	})
}
