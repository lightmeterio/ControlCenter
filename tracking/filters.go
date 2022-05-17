// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"encoding/json"
	"regexp"

	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type Filters []Filter

var NoFilters = Filters{}

func (filters Filters) Reject(r Result) bool {
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

type FilterDescription struct {
	AcceptOutboundSender    string
	AcceptInboundRecipient  string
	RejectInboundRecipient  string
	AcceptOutboundMessageID string
	AcceptInReplyTo         string
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
			pattern, err := regexp.Compile(d.AcceptOutboundSender)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyFromOutboundSender{pattern: pattern})
		}

		if len(d.AcceptInboundRecipient) > 0 {
			pattern, err := regexp.Compile(d.AcceptInboundRecipient)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyFromInboundRecipient{pattern: pattern})
		}

		if len(d.RejectInboundRecipient) > 0 {
			pattern, err := regexp.Compile(d.RejectInboundRecipient)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &RejectFromInboundRecipient{pattern: pattern})
		}

		if len(d.AcceptOutboundMessageID) > 0 {
			pattern, err := regexp.Compile(d.AcceptOutboundMessageID)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyOutboundMessageID{Pattern: pattern})
		}

		if len(d.AcceptInReplyTo) > 0 {
			pattern, err := regexp.Compile(d.AcceptInReplyTo)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptInReplyTo{Pattern: pattern})
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
	Filter(r Result) FilterResult
}

func isAnyNone(r Result, keys ...int) bool {
	for _, k := range keys {
		if r[k].IsNone() {
			return true
		}
	}

	return false
}

func patternMatches(p *regexp.Regexp, r Result, localPart, domainPart int) bool {
	// TODO: somehow optimize it not to allocate a new string every time
	return p.MatchString(r[localPart].Text() + "@" + r[domainPart].Text())
}

type AcceptOnlyFromOutboundSender struct {
	pattern *regexp.Regexp
}

func (f *AcceptOnlyFromOutboundSender) Filter(r Result) FilterResult {
	if isAnyNone(r, QueueSenderLocalPartKey, QueueSenderDomainPartKey, ResultMessageDirectionKey) {
		return FilterResultReject
	}

	if MessageDirection(r[ResultMessageDirectionKey].Int64()) != MessageDirectionOutbound {
		return FilterResultUndecided
	}

	if patternMatches(f.pattern, r, QueueSenderLocalPartKey, QueueSenderDomainPartKey) {
		return FilterResultUndecided
	}

	return FilterResultReject
}

type AcceptOnlyFromInboundRecipient struct {
	pattern *regexp.Regexp
}

func (f *AcceptOnlyFromInboundRecipient) Filter(r Result) FilterResult {
	if isAnyNone(r, ResultRecipientLocalPartKey, ResultRecipientDomainPartKey, ResultMessageDirectionKey) {
		return FilterResultReject
	}

	if MessageDirection(r[ResultMessageDirectionKey].Int64()) != MessageDirectionIncoming {
		return FilterResultUndecided
	}

	if patternMatches(f.pattern, r, ResultRecipientLocalPartKey, ResultRecipientDomainPartKey) {
		return FilterResultUndecided
	}

	return FilterResultReject
}

type RejectFromInboundRecipient struct {
	pattern *regexp.Regexp
}

func (f *RejectFromInboundRecipient) Filter(r Result) FilterResult {
	if isAnyNone(r, ResultRecipientLocalPartKey, ResultRecipientDomainPartKey, ResultMessageDirectionKey) {
		return FilterResultReject
	}

	if MessageDirection(r[ResultMessageDirectionKey].Int64()) != MessageDirectionIncoming {
		return FilterResultUndecided
	}

	if patternMatches(f.pattern, r, ResultRecipientLocalPartKey, ResultRecipientDomainPartKey) {
		return FilterResultReject
	}

	return FilterResultUndecided
}

type AcceptOnlyOutboundMessageID struct {
	Pattern *regexp.Regexp
}

func (f *AcceptOnlyOutboundMessageID) Filter(r Result) FilterResult {
	if isAnyNone(r, ResultMessageDirectionKey) {
		return FilterResultUndecided
	}

	if MessageDirection(r[ResultMessageDirectionKey].Int64()) != MessageDirectionOutbound {
		return FilterResultUndecided
	}

	if isAnyNone(r, QueueMessageIDKey) {
		return FilterResultReject
	}

	if f.Pattern.MatchString(r[QueueMessageIDKey].Text()) {
		return FilterResultUndecided
	}

	return FilterResultReject
}

type AcceptInReplyTo struct {
	Pattern *regexp.Regexp
}

func (f *AcceptInReplyTo) Filter(r Result) FilterResult {
	// We handle only inbound replies for now
	if isAnyNone(r, ResultMessageDirectionKey) || MessageDirection(r[ResultMessageDirectionKey].Int64()) != MessageDirectionIncoming {
		return FilterResultUndecided
	}

	references := []string{}

	if !isAnyNone(r, QueueReferencesHeaderKey) {
		if err := json.Unmarshal(r[QueueReferencesHeaderKey].Blob(), &references); err != nil {
			// if it's not valid JSON, this is not my problem...
			return FilterResultUndecided
		}
	}

	if !isAnyNone(r, QueueInReplyToHeaderKey) {
		references = append(references, r[QueueInReplyToHeaderKey].Text())
	}

	if len(references) == 0 {
		return FilterResultUndecided
	}

	for _, reference := range references {
		if f.Pattern.MatchString(reference) {
			return FilterResultUndecided
		}
	}

	return FilterResultReject
}
