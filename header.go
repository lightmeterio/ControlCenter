package parser

import (
	"net"
	"time"

	"gitlab.com/lightmeter/postfix-log-parser/rawparser"
)

func parseMonth(m []byte) time.Month {
	switch string(m) {
	case "Jan":
		return time.January
	case "Feb":
		return time.February
	case "Mar":
		return time.March
	case "Apr":
		return time.April
	case "May":
		return time.May
	case "Jun":
		return time.June
	case "Jul":
		return time.July
	case "Aug":
		return time.August
	case "Sep":
		return time.September
	case "Oct":
		return time.October
	case "Nov":
		return time.November
	case "Dec":
		return time.December
	}

	panic("Invalid Month! " + string(m))
}

type Time struct {
	Month  time.Month
	Day    uint8
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
	Time      Time
	Host      string
	Process   string
	Daemon    string
	PID       int
	ProcessIP net.IP
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

	pid, err := func() (int, error) {
		if len(h.ProcessID) == 0 {
			return 0, nil
		}

		return atoi(h.ProcessID)
	}()

	if err != nil {
		return Header{}, err
	}

	return Header{
		Time: Time{
			Day:    uint8(day),
			Month:  parseMonth(h.Month),
			Hour:   uint8(hour),
			Minute: uint8(minute),
			Second: uint8(second),
		},
		Host:      string(h.Host),
		Process:   string(h.Process),
		Daemon:    string(h.Daemon),
		PID:       pid,
		ProcessIP: net.ParseIP(string(h.ProcessIP)),
	}, nil
}
