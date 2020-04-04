package parser

import (
	//"errors"
	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"time"
)

func parseMonth(m []byte) time.Month {
	switch string(m) {
	case "Jan":
		return 1
	case "Feb":
		return 2
	case "Mar":
		return 3
	case "Apr":
		return 4
	case "May":
		return 5
	case "Jun":
		return 6
	case "Jul":
		return 7
	case "Aug":
		return 8
	case "Sep":
		return 9
	case "Oct":
		return 10
	case "Nov":
		return 11
	case "Dec":
		return 12
	}

	panic("Invalid Month! " + string(m))
}

type Time struct {
	Day    uint8
	Month  time.Month
	Hour   uint8
	Minute uint8
	Second uint8
}

// Follows convention from time.Data():
//   The month, day, hour, min, sec, and nsec values may be outside their usual ranges
//   and will be normalized during the conversion. For example, October 32 converts to November 1.
func (t Time) Unix(year int, tz *time.Location) int64 {
	return t.Time(year, tz).Unix()
}

func (t Time) Time(year int, tz *time.Location) time.Time {
	return time.Date(year, t.Month, int(t.Day), int(t.Hour), int(t.Minute), int(t.Second), 0, tz)
}

type Header struct {
	Time    Time
	Host    string
	Process string
}

func parseHeader(h rawparser.RawHeader) (Header, error) {
	day, err := atoi(h.Day)

	if err != nil {
		return Header{}, err
	}

	hour, err := atoi(h.Hour)

	if err != nil {
		return Header{}, err
	}

	minute, err := atoi(h.Minute)

	if err != nil {
		return Header{}, err
	}

	second, err := atoi(h.Second)

	if err != nil {
		return Header{}, err
	}

	process := string(h.Process)

	return Header{
		Time: Time{
			Day:    uint8(day),
			Month:  parseMonth(h.Month),
			Hour:   uint8(hour),
			Minute: uint8(minute),
			Second: uint8(second),
		},
		Host:    string(h.Host),
		Process: process,
	}, nil
}
