// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package deliverydb

import (
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/emailutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
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
		default:
			continue
		}
	}

	return false
}

var SettingsKey = `delivery_filters`

type FilterDescription struct {
	AcceptSender    string
	RejectRecipient string
}

// FIXME: meh! we read this description from the default settings, which use Mergo.
// Mergo turns out not to support arrays ([]struct...) which forces us to use this ugly struct
type FiltersDescription struct {
	Rule1 FilterDescription
	Rule2 FilterDescription
	Rule3 FilterDescription
	Rule4 FilterDescription
	Rule5 FilterDescription
	Rule6 FilterDescription
	Rule7 FilterDescription
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
	} {
		if len(d.AcceptSender) > 0 {
			localPart, domainPart, err := emailutil.Split(d.AcceptSender)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &AcceptOnlyFromSender{LocalPart: localPart, DomainPart: domainPart})
		}

		if len(d.RejectRecipient) > 0 {
			localPart, domainPart, err := emailutil.Split(d.RejectRecipient)
			if err != nil {
				return nil, errorutil.Wrap(err)
			}

			filters = append(filters, &RejectFromRecipient{LocalPart: localPart, DomainPart: domainPart})
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

type AcceptOnlyFromSender struct {
	LocalPart  string
	DomainPart string
}

func (f *AcceptOnlyFromSender) Filter(r tracking.Result) FilterResult {
	if r[tracking.QueueSenderLocalPartKey].Text() == f.LocalPart && r[tracking.QueueSenderDomainPartKey].Text() == f.DomainPart {
		return FilterResultUndecided
	}

	return FilterResultReject
}

type RejectFromRecipient struct {
	LocalPart  string
	DomainPart string
}

func (f *RejectFromRecipient) Filter(r tracking.Result) FilterResult {
	if r[tracking.ResultRecipientLocalPartKey].Text() == f.LocalPart && r[tracking.ResultRecipientDomainPartKey].Text() == f.DomainPart {
		return FilterResultReject
	}

	return FilterResultUndecided
}
