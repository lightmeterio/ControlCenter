// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package logeater

import (
	"bufio"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/data"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"io"
	"time"
)

func ReadFromReader(reader io.Reader, pub data.Publisher, ts time.Time) {
	lineNo := uint64(0)

	scanner := bufio.NewScanner(reader)

	converter := parser.NewTimeConverter(ts, func(int, parser.Time, parser.Time) {})

	for {
		if !scanner.Scan() {
			break
		}

		lineNo++

		tryToParseAndPublish(scanner.Bytes(), pub, &converter, lineNo)
	}
}

func tryToParseAndPublish(line []byte, publisher data.Publisher, converter *parser.TimeConverter, lineNo uint64) {
	h, p, err := parser.Parse(line)

	if !parser.IsRecoverableError(err) {
		log.Info().Msgf("Invalid Postfix header: \"%s\"", string(line))
		return
	}

	publisher.Publish(data.Record{
		Time:    converter.Convert(h.Time),
		Header:  h,
		Payload: p,
		Location: data.RecordLocation{
			Line:     lineNo,
			Filename: "unknown",
		},
	})
}

func ParseLogsFromReader(publisher data.Publisher, ts time.Time, reader io.Reader) {
	ReadFromReader(reader, publisher, ts)
}

func BuildInitialLogsTime(mostRecentLogTime time.Time, logYear int) time.Time {
	if !mostRecentLogTime.IsZero() {
		return mostRecentLogTime
	}

	return time.Date(logYear, time.January, 1, 0, 0, 0, 0, time.UTC)
}
