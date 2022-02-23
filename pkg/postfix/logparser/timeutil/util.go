// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timeutil

import (
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

type TimeConverter struct {
	timezone *time.Location
	lastTime Time
	year     int
	clock    timeutil.Clock

	// every time a year change is detected, notifies it
	newYearNotifier func(newYear int, old Time, new Time)
}

func NewTimeConverter(initialTime time.Time, clock timeutil.Clock,
	newYearNotifier func(int, Time, Time),
) TimeConverter {
	return TimeConverter{
		timezone: initialTime.Location(),
		year:     initialTime.Year(),
		lastTime: Time{
			// NOTE; Year is intentionally left out, as we are aiming to figure it out
			Year:   0,
			Month:  initialTime.Month(),
			Day:    uint8(initialTime.Day()),
			Hour:   uint8(initialTime.Hour()),
			Minute: uint8(initialTime.Minute()),
			Second: uint8(initialTime.Second()),
		},
		newYearNotifier: newYearNotifier,
		clock:           clock,
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
	const somehowLessThanOneHour = int64((time.Minute * 50) / time.Second)

	dstChanged := diffInSeconds <= oneHour && diffInSeconds >= somehowLessThanOneHour

	defer func() {
		c.lastTime = t
	}()

	updatedYear := c.year

	if isOutOfOrder && !dstChanged {
		updatedYear++
	}

	newTime := t.Time(updatedYear, c.timezone)

	// We are forbidden to handle logs from the future!
	// See issue #644 for the context
	if newTime.Year() > c.clock.Now().Year() {
		return t.Time(c.year, c.timezone)
	}

	if c.year < updatedYear {
		c.newYearNotifier(updatedYear, c.lastTime, t)
	}

	c.year = updatedYear

	return newTime
}
