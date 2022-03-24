// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	. "github.com/smartystreets/goconvey/convey"
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
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("accept_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("reject_recipient1"),
				ResultRecipientDomainPartKey: ResultEntryText("example2.com"),
			}.Result()), ShouldBeTrue)

			// only sender is checked, as it's outbound
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("accept_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("recipient1"),
				ResultRecipientDomainPartKey: ResultEntryText("example2.com"),
				ResultMessageDirectionKey:    ResultEntryInt64(int64(MessageDirectionOutbound)),
			}.Result()), ShouldBeFalse)

			// only sender is checked, as it's outbound
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("reject_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example2.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("recipient1"),
				ResultRecipientDomainPartKey: ResultEntryText("example2.com"),
				ResultMessageDirectionKey:    ResultEntryInt64(int64(MessageDirectionOutbound)),
			}.Result()), ShouldBeTrue)

			// only recipient is checked, as it's inbound
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("any_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("accept_recipient1"),
				ResultRecipientDomainPartKey: ResultEntryText("example2.com"),
				ResultMessageDirectionKey:    ResultEntryInt64(int64(MessageDirectionIncoming)),
			}.Result()), ShouldBeFalse)

			// only recipient is checked, as it's inbound
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("any_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("reject_recipient"),
				ResultRecipientDomainPartKey: ResultEntryText("example3.com"),
				ResultMessageDirectionKey:    ResultEntryInt64(int64(MessageDirectionIncoming)),
			}.Result()), ShouldBeTrue)
		})

		Convey("Reject inbound recipient", func() {
			filters, err := BuildFilters(FiltersDescription{
				Rule1: FilterDescription{RejectInboundRecipient: "reject_recipient@example1.com"},
			})

			So(err, ShouldBeNil)

			// Missing Direction
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("accept_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("reject_recipient1"),
				ResultRecipientDomainPartKey: ResultEntryText("example2.com"),
			}.Result()), ShouldBeTrue)

			// Outbound Message, so nothing is checked
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("accept_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("reject_recipient"), // not checked, as it's inbound
				ResultRecipientDomainPartKey: ResultEntryText("example1.com"),     // not checked, as it's inbound
				ResultMessageDirectionKey:    ResultEntryInt64(int64(MessageDirectionOutbound)),
			}.Result()), ShouldBeFalse)

			// Inbound Message rejected
			So(filters.Reject(MappedResult{
				QueueSenderLocalPartKey:      ResultEntryText("accept_sender"),
				QueueSenderDomainPartKey:     ResultEntryText("example1.com"),
				ResultRecipientLocalPartKey:  ResultEntryText("reject_recipient"),
				ResultRecipientDomainPartKey: ResultEntryText("example1.com"),
				ResultMessageDirectionKey:    ResultEntryInt64(int64(MessageDirectionIncoming)),
			}.Result()), ShouldBeTrue)
		})

		Convey("By message ID", func() {
			filters, err := BuildFilters(FiltersDescription{
				Rule4: FilterDescription{AcceptOutboundMessageID: `\.(example\.com|otherwise\.de)$`},
			})

			So(err, ShouldBeNil)

			So(filters.Reject(MappedResult{
				QueueMessageIDKey:         ResultEntryText("h6765hhjhg.example.com"),
				ResultMessageDirectionKey: ResultEntryInt64(int64(MessageDirectionOutbound)),
			}.Result()), ShouldBeFalse)

			So(filters.Reject(MappedResult{
				ResultMessageDirectionKey: ResultEntryInt64(int64(MessageDirectionOutbound)),
				QueueMessageIDKey:         ResultEntryText("lalala@somethingelse.net"),
			}.Result()), ShouldBeTrue)

			// message-id missing
			So(filters.Reject(MappedResult{
				ResultMessageDirectionKey: ResultEntryInt64(int64(MessageDirectionOutbound)),
			}.Result()), ShouldBeTrue)
		})
	})
}
