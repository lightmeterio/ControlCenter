// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/dashboard"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/dirlogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/logeater/transform"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"gitlab.com/lightmeter/controlcenter/workspace"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

func simpleDashboard(d dashboard.Dashboard) {
	i, _ := timeutil.ParseTimeInterval("0000-01-01", "5000-01-01", time.UTC)

	for {
		s, _ := d.DeliveryStatus(context.Background(), i)
		fmt.Println(s)
		time.Sleep(time.Second * 1)
	}
}

func main() {
	lmsqlite3.Initialize(lmsqlite3.Options{})

	var (
		workspaceDir   string
		inputFile      string
		inputDirectory string
	)

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")

	flag.StringVar(&workspaceDir, "workspace", "", "path to the workspace")
	flag.StringVar(&inputFile, "file", "", "file to read")
	flag.StringVar(&inputDirectory, "dir", "", "read from a log directory instead")

	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}) //.With().Caller().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

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
	errorutil.MustSucceed(os.MkdirAll(workspaceDir, os.ModePerm))

	ws, err := workspace.NewWorkspace(workspaceDir)
	errorutil.MustSucceed(err)

	importAnnouncer, err := ws.ImportAnnouncer()
	errorutil.MustSucceed(err)

	logSource, err := func() (logsource.Source, error) {
		mostRecentTime, err := ws.MostRecentLogTime()
		errorutil.MustSucceed(err)

		if len(inputDirectory) > 0 {
			return dirlogsource.New(inputDirectory, mostRecentTime, importAnnouncer, false, false, "default", dirwatcher.DefaultLogPatterns)
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

		return filelogsource.New(f, builder, importAnnouncer)
	}()

	errorutil.MustSucceed(err)

	done, cancel := runner.Run(ws)

	pub := ws.NewPublisher()

	logReader := logsource.NewReader(logSource, pub)

	//go simpleDashboard(ws.Dashboard())

	err = logReader.Run()

	errorutil.MustSucceed(err)

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
