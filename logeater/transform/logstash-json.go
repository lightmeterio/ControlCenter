// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	"encoding/json"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

type logstashJsonTransformer struct {
	lineNo uint64
}

func (t *logstashJsonTransformer) Transform(line []byte) (postfix.Record, error) {
	var payload struct {
		Time time.Time `json:"@timestamp"`
		Log  *struct {
			File struct {
				Path string `json:"path"`
			} `json:"file"`
		} `json:"log"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(line, &payload); err != nil {
		return postfix.Record{}, errorutil.Wrap(err)
	}

	lineNo := t.lineNo
	t.lineNo++

	loc := postfix.RecordLocation{
		Line: lineNo,
		Filename: func() string {
			if payload.Log != nil {
				return payload.Log.File.Path
			}

			return "unknown"
		}(),
	}

	r, err := ParseLine([]byte(payload.Message), func(parser.Header) time.Time {
		return payload.Time
	}, loc)
	if err != nil {
		return postfix.Record{}, errorutil.Wrap(err)
	}

	return r, nil
}

func init() {
	Register("logstash", ForwardArgs, func(args ...interface{}) (Transformer, error) {
		return &logstashJsonTransformer{lineNo: 1}, nil
	})
}
