package parser

import (
	"time"
)

type TimeConverter struct {
	timezone *time.Location
	lastTime Time
	year     int

	// every time a year change is detected, notifies it
	newYearNotifier func(newYear int, old Time, new Time)
}

func NewTimeConverter(initialTime time.Time,
	newYearNotifier func(int, Time, Time),
) TimeConverter {
	return TimeConverter{
		timezone: initialTime.Location(),
		year:     initialTime.Year(),
		lastTime: Time{
			Month:  initialTime.Month(),
			Day:    uint8(initialTime.Day()),
			Hour:   uint8(initialTime.Hour()),
			Minute: uint8(initialTime.Minute()),
			Second: uint8(initialTime.Second()),
		},
		newYearNotifier: newYearNotifier,
	}
}

func DefaultTimeInYear(year int, tz *time.Location) time.Time {
	return time.Date(year, time.January, 1, 0, 0, 0, 0, tz)
}

func (c *TimeConverter) Convert(t Time) time.Time {
	diffInSeconds := c.lastTime.Unix(c.year, c.timezone) - t.Unix(c.year, c.timezone)

	// Sometimes log lines are out of order (normally when created by different processes)
	// and we should support such cases.
	// such constant was chosen arbitrarily.
	const maxOutOfOrderOffset = 5

	isOutOfOrder := diffInSeconds > maxOutOfOrderOffset

	oneHour := int64((time.Hour / time.Second))

	// gives the converter a 10min tolerance for then the clock changes one hour backwards
	// NOTE: this should work for most e-mail servers.
	// This is a totally arbitrarily chosen number, though. It has to be something less than
	// one hour, but is not expected to be too small, as I believe most servers are expected to
	// output at least one log line every 10min.
	somehowLessThanOneHour := int64((time.Minute * 50) / time.Second)

	dstChanged := diffInSeconds <= oneHour && diffInSeconds >= somehowLessThanOneHour

	if isOutOfOrder && !dstChanged {
		c.year++
		c.newYearNotifier(c.year, c.lastTime, t)
	}

	c.lastTime = t

	return t.Time(c.year, c.timezone)
}
