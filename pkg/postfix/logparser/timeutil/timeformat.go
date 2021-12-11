// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timeutil

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidTimeFormat = errors.New(`Invalid time format`)

func parseMonth(m string) (time.Month, error) {
	switch m {
	case "Jan":
		return time.January, nil
	case "Feb":
		return time.February, nil
	case "Mar":
		return time.March, nil
	case "Apr":
		return time.April, nil
	case "May":
		return time.May, nil
	case "Jun":
		return time.June, nil
	case "Jul":
		return time.July, nil
	case "Aug":
		return time.August, nil
	case "Sep":
		return time.September, nil
	case "Oct":
		return time.October, nil
	case "Nov":
		return time.November, nil
	case "Dec":
		return time.December, nil
	}

	return time.January, ErrInvalidTimeFormat
}

func atoi(s string) (int, error) {
	return strconv.Atoi(s)
}

type RawTime struct {
	Time   string
	Month  string
	Day    string
	Hour   string
	Minute string
	Second string

	// optional value, available in some syslog configurations
	Year string
}

type Time struct {
	Month  time.Month
	Year   uint16
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

type TimeFormat interface {
	ExtractRaw(string) (h RawTime, remaining string, patternLen int, err error)
	Convert(RawTime) (Time, error)
	ConvertWithYear(Time, int, *time.Location) time.Time
	ConvertWithConverter(*TimeConverter, Time) time.Time
}

var Formats = map[string]TimeFormat{}

func Register(name string, format TimeFormat) {
	Formats[name] = format
}

var ErrInvalidFormat = errors.New(`Invalid Format`)

func Get(name string) (TimeFormat, error) {
	if f, ok := Formats[name]; ok {
		return f, nil
	}

	return nil, ErrInvalidFormat
}

type DefaultTimeFormat struct{}

func init() {
	Register("default", &DefaultTimeFormat{})
}

func (DefaultTimeFormat) ExtractRaw(logLine string) (RawTime, string, int, error) {
	// A line starts with a time, with fixed length
	// the `day` field is always trailed with a space, if needed
	// so it's always two characters long
	const defaultSampleLogDateTime = `Mar 22 06:28:55 `

	if len(logLine) < len(defaultSampleLogDateTime) {
		return RawTime{}, "", 0, ErrInvalidTimeFormat
	}

	remainingHeader := logLine[len(defaultSampleLogDateTime):]

	if len(remainingHeader) == 0 {
		return RawTime{}, "", 0, ErrInvalidTimeFormat
	}

	h := RawTime{
		Time:   logLine[:len(defaultSampleLogDateTime)-1],
		Month:  logLine[0:3],
		Day:    logLine[4:6],
		Hour:   logLine[7:9],
		Minute: logLine[10:12],
		Second: logLine[13:15],
		// Other fields intentionally left empty
	}

	return h, remainingHeader, len(defaultSampleLogDateTime), nil
}

func (DefaultTimeFormat) ConvertWithYear(t Time, year int, tz *time.Location) time.Time {
	return t.Time(year, tz)
}

func (DefaultTimeFormat) ConvertWithConverter(converter *TimeConverter, t Time) time.Time {
	return converter.Convert(t)
}

func (DefaultTimeFormat) Convert(h RawTime) (Time, error) {
	day, err := atoi(strings.TrimLeft(h.Day, ` `))
	if err != nil {
		return Time{}, err
	}

	hour, err := atoi(h.Hour)
	if err != nil {
		return Time{}, err
	}

	minute, err := atoi(h.Minute)
	if err != nil {
		return Time{}, err
	}

	second, err := atoi(h.Second)
	if err != nil {
		return Time{}, err
	}

	month, err := parseMonth(h.Month)
	if err != nil {
		return Time{}, err
	}

	return Time{
		Month:  month,
		Day:    uint8(day),
		Hour:   uint8(hour),
		Minute: uint8(minute),
		Second: uint8(second),
	}, nil
}

type RFC3339TimeFormat struct{}

func init() {
	Register("rfc3339", &RFC3339TimeFormat{})
}

func (RFC3339TimeFormat) ExtractRaw(logLine string) (RawTime, string, int, error) {
	const sampleTime = `2021-05-16T00:01:42.278515+02:00 `
	//                  0123456789012345678901234567890123

	if len(logLine) < len(sampleTime) {
		return RawTime{}, "", 0, ErrInvalidTimeFormat
	}

	remainingHeader := logLine[len(sampleTime):]

	if len(remainingHeader) == 0 {
		return RawTime{}, "", 0, ErrInvalidTimeFormat
	}

	h := RawTime{
		Time:   logLine[:len(sampleTime)-1],
		Year:   logLine[0:4],
		Month:  logLine[5:7],
		Day:    logLine[8:10],
		Hour:   logLine[11:13],
		Minute: logLine[14:16],
		Second: logLine[17:19],
		// Other fields intentionally left empty
	}

	return h, remainingHeader, len(sampleTime), nil
}

func (RFC3339TimeFormat) ConvertWithYear(t Time, _ int, tz *time.Location) time.Time {
	return t.Time(int(t.Year), tz)
}

func (RFC3339TimeFormat) ConvertWithConverter(converter *TimeConverter, t Time) time.Time {
	return t.Time(int(t.Year), converter.timezone)
}

func (RFC3339TimeFormat) Convert(h RawTime) (Time, error) {
	day, err := atoi(strings.TrimLeft(h.Day, ` `))
	if err != nil {
		return Time{}, err
	}

	hour, err := atoi(h.Hour)
	if err != nil {
		return Time{}, err
	}

	minute, err := atoi(h.Minute)
	if err != nil {
		return Time{}, err
	}

	second, err := atoi(h.Second)
	if err != nil {
		return Time{}, err
	}

	monthInt, err := atoi(h.Month)
	if err != nil {
		return Time{}, err
	}

	if monthInt < 1 || monthInt > 12 {
		return Time{}, ErrInvalidTimeFormat
	}

	month := time.Month(monthInt)

	year, err := atoi(h.Year)
	if err != nil {
		return Time{}, err
	}

	return Time{
		Month:  month,
		Day:    uint8(day),
		Hour:   uint8(hour),
		Minute: uint8(minute),
		Second: uint8(second),
		Year:   uint16(year),
	}, nil
}
