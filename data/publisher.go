// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package data

import (
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"time"
)

type RecordLocation struct {
	Line     uint64
	Filename string
}

type Record struct {
	Time     time.Time
	Header   parser.Header
	Location RecordLocation
	Payload  parser.Payload
}

type Publisher interface {
	Publish(Record)
}

type ComposedPublisher []Publisher

func (c ComposedPublisher) Publish(r Record) {
	for _, p := range c {
		p.Publish(r)
	}
}
