// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package localrbl

import (
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"io"
	"time"
)

type Options struct {
	NumberOfWorkers  int
	RBLProvidersURLs []string
	Lookup           DNSLookupFunction
}

type ContentElement struct {
	RBL  string `json:"rbl"`
	Text string `json:"text"`
}

type Results struct {
	Err      error
	Interval data.TimeInterval
	RBLs     []ContentElement
}

type Checker interface {
	io.Closer
	globalsettings.IPAddressGetter
	StartListening()
	NotifyNewScan(time.Time)
	Step(time.Time, func(Results) error, func() error) error
}
