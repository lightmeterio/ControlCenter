package data

import (
	"errors"
	"time"
)

var (
	OutOfOrderTimeInterval = errors.New("Time Interval is Out of Order")
)

type TimeInterval struct {
	From time.Time
	To   time.Time
}

func ParseTimeInterval(fromStr string, toStr string, location *time.Location) (TimeInterval, error) {
	from, err := time.ParseInLocation("2006-01-02", fromStr, location)

	if err != nil {
		return TimeInterval{}, err
	}

	to, err := time.ParseInLocation("2006-01-02", toStr, location)

	if err != nil {
		return TimeInterval{}, err
	}

	to = to.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)

	if from.After(to) {
		return TimeInterval{}, OutOfOrderTimeInterval
	}

	return TimeInterval{From: from, To: to}, nil
}
