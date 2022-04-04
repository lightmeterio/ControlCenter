// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"gitlab.com/lightmeter/controlcenter/deliverydb"
	"gitlab.com/lightmeter/controlcenter/domainmapping"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/tracking"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
)

// Implements an example program that gets an input the output of the `mailtracking`
// example program (txt file where each line is a json encoded tracking.Result).
// The file is received with `-file /path/to/file`. To read from stdin, `-file /dev/stdin`

func main() {
	lmsqlite3.Initialize(lmsqlite3.Options{})

	var (
		workspace string
		inputFile string
	)

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&workspace, "workspace", "", "path to the workspace")
	flag.StringVar(&inputFile, "file", "", "file to read")

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

	f, err := os.Open(inputFile)
	errorutil.MustSucceed(err)

	// ensure workspace exists
	errorutil.MustSucceed(os.MkdirAll(workspace, os.ModePerm))

	dbFilename := path.Join(workspace, "logs.db")
	conn, err := dbconn.Open(dbFilename, 10)
	errorutil.MustSucceed(err)

	defer func() {
		errorutil.MustSucceed(conn.Close())
	}()

	err = migrator.Run(conn.RwConn.DB, "logs")
	errorutil.MustSucceed(err)

	db, err := deliverydb.New(conn, &domainmapping.DefaultMapping, tracking.NoFilters)
	errorutil.MustSucceed(err)

	pub := db.ResultsPublisher()

	done, cancel := runner.Run(db)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		var result tracking.Result
		err = json.Unmarshal(scanner.Bytes(), &result)
		errorutil.MustSucceed(err)
		pub.Publish(result)
	}

	cancel()
	done()

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
