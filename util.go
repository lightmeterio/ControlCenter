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

func (this *TimeConverter) Convert(t Time) time.Time {
	// Bump the year if we read something that looks like going backwards
	// This is not a clever way to do it and can lead to many issues
	// (one second backward will move to the next year!),
	// but it works for now. A better way is inform the sysadmin about
	// inconsistencies in the logs instead
	// FIXME: this probably won't work for any time before the unix epoch (ts negative),
	// but who will go back in time and run Postfix?
	if this.lastTime.Unix(this.year, this.timezone) > t.Unix(this.year, this.timezone) {
		this.year++
		this.newYearNotifier(this.year, this.lastTime, t)
	}

	this.lastTime = t
	return t.Time(this.year, this.timezone)
}
