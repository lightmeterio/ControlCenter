// SPDX-FileCopyrightText: 2020,  Lightmeter <hello@lightmeter.io>
// SPDX-License-Identifier: AGPL-3.0

package testutil

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TempDir(t *testing.T) (string, func()) {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")

	if e != nil {
		panic("error creating temp dir")
	}

	return dir, func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal("Could not remove tempdir", dir, "error:", err)
		}
	}
}

// MustParseTime parses a time in the format `2006-01-02 15:04:05 -0700`
// and panics in case the parsing fails
func MustParseTime(s string) time.Time {
	p, err := time.Parse(`2006-01-02 15:04:05 -0700`, s)

	if err != nil {
		panic("parsing time: " + err.Error())
	}

	return p.In(time.UTC)
}
