package parser

import (
	"time"
)

type TimeConverterParser struct {
	year      int
	tz        *time.Location
	converter TimeConverter
	notifier  func(year int, before, after Time)
}

func NewTimeConverterParser(year int, tz *time.Location, notifier func(year int, before, after Time)) *TimeConverterParser {
	return &TimeConverterParser{year, tz, NewTimeConverter(Time{}, year, tz, notifier), notifier}
}

func (c *TimeConverterParser) Parse(line []byte) (time.Time, Header, Payload, error) {
	h, p, err := Parse(line)

	if !IsRecoverableError(err) {
		return time.Time{}, Header{}, nil, err
	}

	t := c.converter.Convert(h.Time)

	return t, h, p, err
}
