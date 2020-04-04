package postfix

import (
	parser "gitlab.com/lightmeter/postfix-log-parser"
	"log"
	"time"
)

type TimeConverter struct {
	timezone       *time.Location
	lastTime       parser.Time
	year           int
	firstExecution bool
}

func NewTimeConverter(initialTime parser.Time, year int, timezone *time.Location) TimeConverter {
	return TimeConverter{
		firstExecution: true,
		timezone:       timezone,
		year:           year,
		lastTime:       initialTime,
	}
}

func (this *TimeConverter) Convert(t parser.Time) time.Time {
	// Bump the year if we read something that looks like going backwards
	// This is not a clever way to do it and can lead to many issues
	// (one second backward will move to the next year!),
	// but it works for now. A better way is inform the sysadmin about
	// inconsistencies in the logs instead
	// FIXME: this probably won't work for any time before the unix epoch (ts negative),
	// but who will go back in time and run Postfix?
	if this.lastTime.Unix(this.year, this.timezone) > t.Unix(this.year, this.timezone) {
		log.Println("Bumping year", this.year, ", old:", this.lastTime, ", new:", t)
		this.year += 1
	}

	this.firstExecution = false
	this.lastTime = t
	return t.Time(this.year, this.timezone)
}
