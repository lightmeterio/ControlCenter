// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package transform

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	parsertimeutil "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"time"
)

// A Transformer parses a Pogtfix log line which might be embedded into other formats
type Transformer interface {
	Transform(string) (postfix.Record, error)
}

type getter struct {
	argsBuilder func(args ...interface{}) ([]interface{}, error)
	builder     func(args ...interface{}) (Transformer, error)
}

var getters = map[string]getter{}

func Register(name string, argsBuilder func(args ...interface{}) ([]interface{}, error), builder func(args ...interface{}) (Transformer, error)) {
	getters[name] = getter{argsBuilder: argsBuilder, builder: builder}
}

var ErrUnknownTransformer = errors.New(`Unknown Format`)

type Builder func() (Transformer, error)

func Get(name string, args ...interface{}) (Builder, error) {
	getter, ok := getters[name]
	if !ok {
		return nil, ErrUnknownTransformer
	}

	builtArgs, err := getter.argsBuilder(args...)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return func() (Transformer, error) {
		return getter.builder(builtArgs...)
	}, nil
}

type defaultTransformer struct {
	converter parsertimeutil.TimeConverter
	lineNo    uint64
}

func ParseLine(line string, timeBuilder func(parser.Header) time.Time, loc postfix.RecordLocation, format parsertimeutil.TimeFormat) (postfix.Record, error) {
	h, p, err := parser.ParseWithCustomTimeFormat(line, format)

	if !parser.IsRecoverableError(err) {
		return postfix.Record{}, errorutil.Wrap(err, loc)
	}

	time := timeBuilder(h)

	return postfix.Record{
		Time:     time,
		Header:   h,
		Payload:  p,
		Location: loc,
		Line:     line,
	}, nil
}

func ForwardArgs(args ...interface{}) ([]interface{}, error) {
	return args, nil
}

var defaultTimeFormat = parsertimeutil.DefaultTimeFormat{}

func (t *defaultTransformer) Transform(line string) (postfix.Record, error) {
	lineNo := t.lineNo
	t.lineNo++

	loc := postfix.RecordLocation{
		Line:     lineNo,
		Filename: "unknown",
	}

	r, err := ParseLine(line, func(h parser.Header) time.Time {
		return t.converter.Convert(h.Time)
	}, loc, defaultTimeFormat)
	if err != nil {
		return postfix.Record{}, errorutil.Wrap(err)
	}

	return r, nil
}

func init() {
	Register("default", func(args ...interface{}) ([]interface{}, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("At least one argument is needed")
		}

		clock, ok := args[0].(timeutil.Clock)
		if !ok {
			return nil, fmt.Errorf("A clock is needed on registering")
		}

		defaultYear := func() int { return clock.Now().Year() }

		if len(args) == 1 {
			return []interface{}{clock, defaultYear}, nil
		}

		year, ok := args[1].(int)

		if !ok {
			return nil, fmt.Errorf("Argument is not a valid year: %v", args[1])
		}

		if year == 0 {
			return []interface{}{clock, defaultYear}, nil
		}

		return []interface{}{clock, func() int { return year }}, nil
	}, func(args ...interface{}) (Transformer, error) {
		//nolint:forcetypeassert
		clock, ok := args[0].(timeutil.Clock)
		if !ok {
			return nil, fmt.Errorf("A clock is needed on building")
		}

		//nolint:forcetypeassert
		getYear := args[1].(func() int)

		initialTime := time.Date(getYear(), time.January, 1, 0, 0, 0, 0, time.UTC)

		return &defaultTransformer{
			lineNo: 1,
			converter: parsertimeutil.NewTimeConverter(initialTime, clock, func(year int, previousTime parser.Time, newTime parser.Time) {
				log.Info().Msgf("Year changed to %v", year)
			}),
		}, nil
	})
}
