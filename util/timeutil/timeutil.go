// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timeutil

import (
	"fmt"
	"time"
)

// MustParseTime parses a time in the format `2006-01-02 15:04:05 -0700`
// and panics in case the parsing fails
func MustParseTime(s string) time.Time {
	p, err := time.Parse(`2006-01-02 15:04:05 -0700`, s)

	if err != nil {
		panic("parsing time: " + err.Error())
	}

	return p.In(time.UTC)
}

// TODO: remove this function (see gitlab issue #259)
func PrettyFormatTime(time time.Time, language string) string {
	switch language {
	case "en":
		return time.Format("02 Jan. (3:00PM)")
	case "de":
		return Format(time, deMonths)
	case "pt_BR":
		return Format(time, ptBrMonths)
	}

	// fallback
	return time.Format("02 Jan. (3:00PM)")
}

// TODO: remove this function (see gitlab issue #259)
func Format(t time.Time, months []string) string {
	return fmt.Sprintf("%02d. %s %02d:%02d:%02d",
		t.Day(), months[t.Month()-1][:3], t.Hour(), t.Minute(), t.Second(),
	)
}

var deMonths = []string{
	"Januar", "Februar", "März", "April", "Mai", "Juni",
	"Juli", "August", "September", "Oktober", "November", "Dezember",
}

var ptBrMonths = []string{
	"janeiro", "fevereiro", "março", "abril", "maio", "junho",
	"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
}
