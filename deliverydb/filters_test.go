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
		Convey("Accept only outbound sender and inbound recipient", func() {
			filters, err := BuildFilters(FiltersDescription{
				Rule1: FilterDescription{AcceptOutboundSender: "(accept_sender|another_accepted_sender)@example1.com"},
				Rule2: FilterDescription{AcceptInboundRecipient: "accept_recipient[1234]@example2.com"},
			})

			So(err, ShouldBeNil)

			// Direction missing. Reject.
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient1"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example2.com"),
			}.Result()), ShouldBeTrue)

			// only sender is checked, as it's outbound
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient1"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example2.com"),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			}.Result()), ShouldBeFalse)

			// only sender is checked, as it's outbound
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("reject_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example2.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("recipient1"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example2.com"),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			}.Result()), ShouldBeTrue)

			// only recipient is checked, as it's inbound
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("any_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("accept_recipient1"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example2.com"),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
			}.Result()), ShouldBeFalse)

			// only recipient is checked, as it's inbound
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("any_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example3.com"),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
			}.Result()), ShouldBeTrue)
		})

		Convey("Reject inbound recipient", func() {
			filters, err := BuildFilters(FiltersDescription{
				Rule1: FilterDescription{RejectInboundRecipient: "reject_recipient@example1.com"},
			})

			So(err, ShouldBeNil)

			// Missing Direction
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient1"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example2.com"),
			}.Result()), ShouldBeTrue)

			// Outbound Message, so nothing is checked
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient"), // not checked, as it's inbound
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example1.com"),     // not checked, as it's inbound
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			}.Result()), ShouldBeFalse)

			// Inbound Message rejected
			So(filters.Reject(tracking.MappedResult{
				tracking.QueueSenderLocalPartKey:      tracking.ResultEntryText("accept_sender"),
				tracking.QueueSenderDomainPartKey:     tracking.ResultEntryText("example1.com"),
				tracking.ResultRecipientLocalPartKey:  tracking.ResultEntryText("reject_recipient"),
				tracking.ResultRecipientDomainPartKey: tracking.ResultEntryText("example1.com"),
				tracking.ResultMessageDirectionKey:    tracking.ResultEntryInt64(int64(tracking.MessageDirectionIncoming)),
			}.Result()), ShouldBeTrue)
		})

		Convey("By message ID", func() {
			filters, err := BuildFilters(FiltersDescription{
				Rule4: FilterDescription{AcceptOutboundMessageID: `\.(example\.com|otherwise\.de)$`},
			})

			So(err, ShouldBeNil)

			So(filters.Reject(tracking.MappedResult{
				tracking.QueueMessageIDKey:         tracking.ResultEntryText("h6765hhjhg.example.com"),
				tracking.ResultMessageDirectionKey: tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			}.Result()), ShouldBeFalse)

			So(filters.Reject(tracking.MappedResult{
				tracking.ResultMessageDirectionKey: tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
				tracking.QueueMessageIDKey:         tracking.ResultEntryText("lalala@somethingelse.net"),
			}.Result()), ShouldBeTrue)

			// message-id missing
			So(filters.Reject(tracking.MappedResult{
				tracking.ResultMessageDirectionKey: tracking.ResultEntryInt64(int64(tracking.MessageDirectionOutbound)),
			}.Result()), ShouldBeTrue)
		})
	})
}
