package data

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

var (
	ErrOutOfOrderTimeInterval = errors.New("Time Interval is Out of Order")
)

type TimeInterval struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func ParseTimeInterval(fromStr string, toStr string, location *time.Location) (TimeInterval, error) {
	from, err := time.ParseInLocation("2006-01-02", fromStr, location)

	if err != nil {
		return TimeInterval{}, errorutil.Wrap(err)
	}

	to, err := time.ParseInLocation("2006-01-02", toStr, location)

	if err != nil {
		return TimeInterval{}, errorutil.Wrap(err)
	}

	to = to.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)

	if from.After(to) {
		return TimeInterval{}, ErrOutOfOrderTimeInterval
	}

	return TimeInterval{From: from, To: to}, nil
}
