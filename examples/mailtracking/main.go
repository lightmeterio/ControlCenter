// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/dirlogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

type publisher struct {
}

var counter uint64 = 0

func (*publisher) Publish(r tracking.Result) {
	counter++

	j, err := json.Marshal(r)

	errorutil.MustSucceed(err)

	fmt.Println(string(j))
}

func main() {
	lmsqlite3.Initialize(lmsqlite3.Options{})

	var (
		workspace      string
		inputFile      string
		inputDirectory string
	)

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&workspace, "workspace", "", "path to the workspace")
	flag.StringVar(&inputFile, "file", "", "file to read")
	flag.StringVar(&inputDirectory, "dir", "", "read from a log directory instead")

	flag.Parse()

	// copied from https://golang.org/pkg/runtime/pprof/
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			errorutil.LogFatalf(err, "could not create CPU profile")
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			errorutil.LogFatalf(err, "could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	// ensure workspace exists
	errorutil.MustSucceed(os.MkdirAll(workspace, os.ModePerm))

	logSource, err := func() (logsource.Source, error) {
		if len(inputDirectory) > 0 {
			return dirlogsource.New(inputDirectory, time.Time{}, false, false)
		}

		f, err := os.Open(inputFile)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		year := time.Now().Year()

		builder, err := transform.Get("default", year)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return filelogsource.New(f, builder)
	}()

	errorutil.MustSucceed(err)

	pub := publisher{}

	t, err := tracking.New(workspace, &pub)

	defer func() {
		errorutil.MustSucceed(t.Close())
	}()

	errorutil.MustSucceed(err)

	publisher := t.Publisher()

	logReader := logsource.NewReader(logSource, publisher)

	done, cancel := t.Run()

	err = logReader.Run()

	errorutil.MustSucceed(err)

	cancel()
	done()

	log.Println("Number of messages processed:", counter)

	// copied from https://golang.org/pkg/runtime/pprof/
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			errorutil.LogFatalf(err, "could not create memory profile")
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			errorutil.LogFatalf(err, "could not write memory profile")
		}
	}
}
