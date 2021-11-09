// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

var rfc3339TimeFormat = parsertimeutil.RFC3339TimeFormat{}

type rfc3339Transformer struct {
	lineNo uint64
}

func (t *rfc3339Transformer) Transform(line string) (postfix.Record, error) {
	lineNo := t.lineNo
	t.lineNo++

	loc := postfix.RecordLocation{
		Line:     lineNo,
		Filename: "unknown",
	}

	r, err := ParseLine(line, func(h parser.Header) time.Time {
		return time.Date(int(h.Time.Year), h.Time.Month, int(h.Time.Day), int(h.Time.Hour), int(h.Time.Minute), int(h.Time.Second), 0, time.UTC)
	}, loc, rfc3339TimeFormat)
	if err != nil {
		return postfix.Record{}, errorutil.Wrap(err)
	}

	return r, nil
}

func init() {
	Register("rfc3339", func(args ...interface{}) ([]interface{}, error) {
		return nil, nil
	}, func(args ...interface{}) (Transformer, error) {
		return &rfc3339Transformer{
			lineNo: 1,
		}, nil
	})
}
