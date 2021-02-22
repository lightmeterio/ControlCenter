// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
)

type ResultEntryType int

const (
	ResultEntryTypeNone ResultEntryType = iota
	ResultEntryTypeText
	ResultEntryTypeBlob
	ResultEntryTypeInt64
	ResultEntryTypeFloat64
)

// all possible sqlite types
type ResultEntry struct {
	asText    string
	asBlob    []byte
	asInt64   int64
	asFloat64 float64
	typ       ResultEntryType
}

func (e ResultEntry) ValueOrNil() interface{} {
	switch e.typ {
	case ResultEntryTypeBlob:
		return e.asBlob
	case ResultEntryTypeFloat64:
		return e.asFloat64
	case ResultEntryTypeText:
		return e.asText
	case ResultEntryTypeInt64:
		return e.asInt64
	case ResultEntryTypeNone:
		return nil
	default:
		panic("Invalid type")
	}
}

func (e ResultEntry) Value() interface{} {
	v := e.ValueOrNil()

	if v == nil {
		panic("Invalid type")
	}

	return v
}

func (e ResultEntry) IsNone() bool {
	return e.typ == ResultEntryTypeNone
}

func (e ResultEntry) Int64() int64 {
	if e.typ != ResultEntryTypeInt64 {
		log.Panic().Msgf("Not int64: %v", e)
	}

	return e.asInt64
}

func (e ResultEntry) Float64() float64 {
	if e.typ != ResultEntryTypeFloat64 {
		log.Panic().Msgf("Not float64: %v", e)
	}

	return e.asFloat64
}

func (e ResultEntry) Blob() []byte {
	if e.typ != ResultEntryTypeBlob {
		log.Panic().Msgf("Not blob: %v", e)
	}

	return e.asBlob
}

func (e ResultEntry) Text() string {
	if e.typ != ResultEntryTypeText {
		log.Panic().Msgf("Not text: %v", e)
	}

	return e.asText
}

func ResultEntryText(v string) ResultEntry {
	return ResultEntry{asText: v, typ: ResultEntryTypeText}
}

func ResultEntryBlob(v []byte) ResultEntry {
	return ResultEntry{asBlob: v, typ: ResultEntryTypeBlob}
}

func ResultEntryInt64(v int64) ResultEntry {
	return ResultEntry{asInt64: v, typ: ResultEntryTypeInt64}
}

func ResultEntryFloat64(v float64) ResultEntry {
	return ResultEntry{asFloat64: v, typ: ResultEntryTypeFloat64}
}

func ResultEntryNone() ResultEntry {
	return ResultEntry{typ: ResultEntryTypeNone}
}

func ResultEntryFromValue(i interface{}) ResultEntry {
	switch v := i.(type) {
	case string:
		return ResultEntryText(v)
	case []byte:
		return ResultEntryBlob(v)
	case int64:
		return ResultEntryInt64(v)
	case float64:
		return ResultEntryFloat64(v)
	default:
		return ResultEntryNone()
	}
}

func (e ResultEntry) MarshalJSON() ([]byte, error) {
	t := func() string {
		switch e.typ {
		case ResultEntryTypeBlob:
			return "blob"
		case ResultEntryTypeFloat64:
			return "float64"
		case ResultEntryTypeInt64:
			return "int64"
		case ResultEntryTypeText:
			return "text"
		case ResultEntryTypeNone:
			return "none"
		default:
			panic("invalid type!")
		}
	}()

	return json.Marshal(map[string]interface{}{"type": t, "value": e.ValueOrNil()})
}

type Result [lasResulttKey]ResultEntry

func (r Result) MarshalJSON() ([]byte, error) {
	s := map[string]interface{}{}

	// omit "none" values
	for i, v := range r {
		if !v.IsNone() {
			s[KeysToLabels[i]] = v
		}
	}

	return json.Marshal(s)
}

type ResultPublisher interface {
	Publish(Result)
}
