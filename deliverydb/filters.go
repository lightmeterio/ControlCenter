// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"regexp"
)

type Filters []Filter

var NoFilters = Filters{}

func (filters Filters) Reject(r tracking.Result) bool {
	for _, f := range filters {
		switch f.Filter(r) {
		case FilterResultReject:
			return true
		case FilterResultAccept:
			return false
		case FilterResultUndecided:
			fallthrough
		default:
			continue
		}
	}

	return false
}

var SettingsKey = `delivery_filters`

type FilterDescription struct {
	AcceptOutboundSender    string
	AcceptInboundRecipient  string
	RejectInboundRecipient  string
	AcceptOutboundMessageID string
}

// FIXME: meh! we read this description from the default settings, which use Mergo.
// Mergo turns out not to support arrays ([]struct...) which forces us to use this ugly struct
type FiltersDescription struct {
	Rule1  FilterDescription
	Rule2  FilterDescription
	Rule3  FilterDescription
	Rule4  FilterDescription
	Rule5  FilterDescription
	Rule6  FilterDescription
	Rule7  FilterDescription
	Rule8  FilterDescription
	Rule9  FilterDescription
	Rule10 FilterDescription
	Rule11 FilterDescription
	Rule12 FilterDescription
}

func BuildFilters(desc FiltersDescription) (Filters, error) {
	filters := Filters{}

	for _, d := range []FilterDescription{
		desc.Rule1,
		desc.Rule2,
		desc.Rule3,
		desc.Rule4,
		desc.Rule5,
		desc.Rule6,
		desc.Rule7,
		desc.Rule8,
		desc.Rule9,
		desc.Rule10,
		desc.Rule11,
		desc.Rule12,
	} {
		if len(d.AcceptOutboundSender) > 0 {
			localPart, domainPart, err := emailutil.Split(d.AcceptOutboundSender)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyFromOutboundSender{LocalPart: localPart, DomainPart: domainPart})
		}

		if len(d.AcceptInboundRecipient) > 0 {
			localPart, domainPart, err := emailutil.Split(d.AcceptInboundRecipient)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyFromInboundRecipient{LocalPart: localPart, DomainPart: domainPart})
		}

		if len(d.RejectInboundRecipient) > 0 {
			localPart, domainPart, err := emailutil.Split(d.RejectInboundRecipient)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &RejectFromInboundRecipient{LocalPart: localPart, DomainPart: domainPart})
		}

		if len(d.AcceptOutboundMessageID) > 0 {
			pattern, err := regexp.Compile(d.AcceptOutboundMessageID)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyOutboundMessageID{Pattern: pattern})
		}
	}

	return filters, nil
}

type FilterResult int

const (
	FilterResultAccept FilterResult = iota
	FilterResultReject
	FilterResultUndecided
)

type Filter interface {
	Filter(r tracking.Result) FilterResult
}

func isAnyNone(r tracking.Result, keys ...int) bool {
	for _, k := range keys {
		if r[k].IsNone() {
			return true
		}
	}

	return false
}

type AcceptOnlyFromOutboundSender struct {
	LocalPart  string
	DomainPart string
}

func (f *AcceptOnlyFromOutboundSender) Filter(r tracking.Result) FilterResult {
	if isAnyNone(r, tracking.QueueSenderLocalPartKey, tracking.QueueSenderDomainPartKey, tracking.ResultMessageDirectionKey) {
		return FilterResultReject
	}

	if tracking.MessageDirection(r[tracking.ResultMessageDirectionKey].Int64()) != tracking.MessageDirectionOutbound {
		return FilterResultUndecided
	}

	if r[tracking.QueueSenderLocalPartKey].Text() == f.LocalPart && r[tracking.QueueSenderDomainPartKey].Text() == f.DomainPart {
		return FilterResultUndecided
	}

	return FilterResultReject
}

type AcceptOnlyFromInboundRecipient struct {
	LocalPart  string
	DomainPart string
}

func (f *AcceptOnlyFromInboundRecipient) Filter(r tracking.Result) FilterResult {
	if isAnyNone(r, tracking.ResultRecipientLocalPartKey, tracking.ResultRecipientDomainPartKey, tracking.ResultMessageDirectionKey) {
		return FilterResultReject
	}

	if tracking.MessageDirection(r[tracking.ResultMessageDirectionKey].Int64()) != tracking.MessageDirectionIncoming {
		return FilterResultUndecided
	}

	if r[tracking.ResultRecipientLocalPartKey].Text() == f.LocalPart && r[tracking.ResultRecipientDomainPartKey].Text() == f.DomainPart {
		return FilterResultUndecided
	}

	return FilterResultReject
}

type RejectFromInboundRecipient struct {
	LocalPart  string
	DomainPart string
}

func (f *RejectFromInboundRecipient) Filter(r tracking.Result) FilterResult {
	if isAnyNone(r, tracking.ResultRecipientLocalPartKey, tracking.ResultRecipientDomainPartKey, tracking.ResultMessageDirectionKey) {
		return FilterResultReject
	}

	if tracking.MessageDirection(r[tracking.ResultMessageDirectionKey].Int64()) != tracking.MessageDirectionIncoming {
		return FilterResultUndecided
	}

	if r[tracking.ResultRecipientLocalPartKey].Text() == f.LocalPart && r[tracking.ResultRecipientDomainPartKey].Text() == f.DomainPart {
		return FilterResultReject
	}

	return FilterResultUndecided
}

type AcceptOnlyOutboundMessageID struct {
	Pattern *regexp.Regexp
}

func (f *AcceptOnlyOutboundMessageID) Filter(r tracking.Result) FilterResult {
	if isAnyNone(r, tracking.ResultMessageDirectionKey) {
		return FilterResultUndecided
	}

	if tracking.MessageDirection(r[tracking.ResultMessageDirectionKey].Int64()) != tracking.MessageDirectionOutbound {
		return FilterResultUndecided
	}

	if isAnyNone(r, tracking.QueueMessageIDKey) {
		return FilterResultReject
	}

	if f.Pattern.MatchString(r[tracking.QueueMessageIDKey].Text()) {
		return FilterResultUndecided
	}

	return FilterResultReject
}
