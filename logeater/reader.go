package logeater

import (
	"bufio"
	"io"
	"log"
	"time"

	"gitlab.com/lightmeter/controlcenter/data"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
)

func ReadFromReader(reader io.Reader, pub data.Publisher, ts time.Time) {
	scanner := bufio.NewScanner(reader)

	converter := parser.NewTimeConverter(ts, func(int, parser.Time, parser.Time) {})

	for {
		if !scanner.Scan() {
			break
		}

		tryToParseAndPublish(scanner.Bytes(), pub, &converter)
	}
}

func tryToParseAndPublish(line []byte, publisher data.Publisher, converter *parser.TimeConverter) {
	h, p, err := parser.Parse(line)

	if !parser.IsRecoverableError(err) {
		log.Printf("Invalid Postfix header: \"%s\"", string(line))
		return
	}

	publisher.Publish(data.Record{Time: converter.Convert(h.Time), Header: h, Payload: p})
}

func ParseLogsFromReader(publisher data.Publisher, ts time.Time, reader io.Reader) {
	ReadFromReader(reader, publisher, ts)
}

func BuildInitialLogsTime(mostRecentLogTime time.Time, logYear int, timezone *time.Location) time.Time {
	if !mostRecentLogTime.IsZero() {
		return mostRecentLogTime
	}

	return time.Date(logYear, time.January, 1, 0, 0, 0, 0, timezone)
}
