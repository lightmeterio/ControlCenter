// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	"fmt"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"strings"
	"time"
)

type prependRfc3399Transformer struct {
	lineNo uint64
}

// format: time <space> rawline
// example: 2021-03-06T06:09:00.798Z Mar  6 07:08:59 melian postfix/qmgr[28829]: A1E1E1880093: removed
func (t *prependRfc3399Transformer) Transform(line string) (postfix.Record, error) {
	lineNo := t.lineNo
	t.lineNo++

	loc := postfix.RecordLocation{
		Line:     lineNo,
		Filename: "unknown",
	}

	index := strings.Index(line, string(" "))

	if index == -1 {
		return postfix.Record{}, fmt.Errorf("Error parsing time from line %v", t.lineNo)
	}

	parsedTime, err := time.Parse(time.RFC3339, line[:index])
	if err != nil {
		return postfix.Record{}, errorutil.Wrap(err)
	}

	r, err := ParseLine(line[index+1:], func(parser.Header) time.Time {
		return parsedTime
	}, loc, defaultTimeFormat)
	if err != nil {
		return postfix.Record{}, errorutil.Wrap(err)
	}

	return r, nil
}

func init() {
	Register("prepend-rfc3339", ForwardArgs, func(args ...interface{}) (Transformer, error) {
		return &prependRfc3399Transformer{lineNo: 1}, nil
	})
}
