// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package tracking

import (
	"encoding/base64"
	"encoding/json"
	"github.com/rs/zerolog"
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
		log.Panic().Msg("Invalid type")
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

func resultTypeAsString(e ResultEntry) string {
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
}

func stringAsResultType(s string) ResultEntryType {
	switch s {
	case "blob":
		return ResultEntryTypeBlob
	case "float64":
		return ResultEntryTypeFloat64
	case "int64":
		return ResultEntryTypeInt64
	case "text":
		return ResultEntryTypeText
	case "none":
		return ResultEntryTypeNone
	default:
		panic("invalid type!")
	}
}

func (e ResultEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{"type": resultTypeAsString(e), "value": e.ValueOrNil()})
}

var LabelsToKeys = map[string]int{}

func init() {
	for k, l := range KeysToLabels {
		LabelsToKeys[l] = k
	}
}

func (r *Result) UnmarshalJSON(b []byte) error {
	var m map[string]struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}

	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	for label, v := range m {
		typ := stringAsResultType(v.Type)
		entry := ResultEntry{}
		entry.typ = typ

		switch typ {
		case ResultEntryTypeBlob:
			strValue := `"` + v.Value.(string) + `"`
			length := base64.StdEncoding.DecodedLen(len(strValue))
			entry.asBlob = make([]byte, length)

			if err := json.Unmarshal([]byte(strValue), &entry.asBlob); err != nil {
				return err
			}
		case ResultEntryTypeInt64:
			entry = ResultEntryInt64(int64(v.Value.(float64)))
		case ResultEntryTypeText:
			fallthrough
		case ResultEntryTypeFloat64:
			entry = ResultEntryFromValue(v.Value)
		case ResultEntryTypeNone:
			fallthrough
		default:
		}

		r[LabelsToKeys[label]] = entry
	}

	return nil
}

func (e ResultEntry) MarshalZerologObject(event *zerolog.Event) {
	switch e.typ {
	case ResultEntryTypeBlob:
		event.Bytes("blob", e.asBlob)
	case ResultEntryTypeFloat64:
		event.Float64("float64", e.asFloat64)
	case ResultEntryTypeInt64:
		event.Int64("int64", e.asInt64)
	case ResultEntryTypeText:
		event.Str("text", e.asText)
	case ResultEntryTypeNone:
		event.Bool("none", true)
	default:
		panic("invalid type!")
	}
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

func (r Result) MarshalZerologObject(e *zerolog.Event) {
	// omit "none" values
	for i, v := range r {
		if !v.IsNone() {
			e.Object(KeysToLabels[i], v)
		}
	}
}

type ResultPublisher interface {
	Publish(Result)
}

// Useful for tests, makes it easier to create results
type MappedResult map[ResultEntryType]ResultEntry

func (m MappedResult) Result() Result {
	r := Result{}

	for k, v := range m {
		r[k] = v
	}

	return r
}
