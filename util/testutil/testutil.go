package testutil

import (
	"io/ioutil"
	"time"
)

func TempDir() string {
	dir, e := ioutil.TempDir("", "lightmeter-tests-*")

	if e != nil {
		panic("error creating temp dir")
	}

	return dir
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
