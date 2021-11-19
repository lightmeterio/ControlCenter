// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timeutil

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

func (i TimeInterval) IsZero() bool {
	return i.From.IsZero() && i.To.IsZero()
}

func parseTimeIntervalComponent(s string, location *time.Location) (time.Time, bool, error) {
	const (
		longFormat  = `2006-01-02 15:04:05`
		shortFormat = `2006-01-02`
	)

	if len(s) == len(shortFormat) {
		t, err := time.ParseInLocation(shortFormat, s, location)
		if err != nil {
			return time.Time{}, false, ErrInvalidTime
		}

		return t, true, nil
	}

	if len(s) == len(longFormat) {
		t, err := time.ParseInLocation(longFormat, s, location)
		if err != nil {
			return time.Time{}, false, ErrInvalidTime
		}

		return t, false, nil
	}

	return time.Time{}, false, ErrInvalidTime
}

var ErrInvalidTime = errors.New(`Invalid time`)

func ParseTimeInterval(fromStr string, toStr string, location *time.Location) (TimeInterval, error) {
	from, _, err := parseTimeIntervalComponent(fromStr, location)
	if err != nil {
		return TimeInterval{}, errorutil.Wrap(err)
	}

	to, isShortFormat, err := parseTimeIntervalComponent(toStr, location)
	if err != nil {
		return TimeInterval{}, errorutil.Wrap(err)
	}

	if isShortFormat {
		to = to.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
	}

	if from.After(to) {
		return TimeInterval{}, ErrOutOfOrderTimeInterval
	}

	return TimeInterval{From: from, To: to}, nil
}
