// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

// To profile directory importing with a nice UI, execute:
// ./postfix-dir-watcher -dir /path/to/logs/dir -cpuprofile cpu.out -memprofile mem.out
// go run github.com/google/pprof -http ":6061" mem.out // or cpu.out
// and open your browser on http://localhost:6061

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/dirlogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

type pub struct {
}

func (p *pub) Publish(r postfix.Record) {
	if r.Payload == nil {
		return
	}

	j, err := json.Marshal(map[string]interface{}{
		"header":   r.Header,
		"payload":  r.Payload,
		"filename": r.Location.Filename,
		"line":     r.Location.Line,
	})

	if err != nil {
		log.Panic().Err(err).Msgf("JSON Error")
	}

	fmt.Println(string(j))
}

func main() {
	dirToWatch := flag.String("dir", "", "Directory to watch")

	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile := flag.String("memprofile", "", "write memory profile to `file`")

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

	if len(*dirToWatch) == 0 {
		log.Fatal().Msg("-dir is mandatory!")
	}

	logSource, err := dirlogsource.New(*dirToWatch, time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC), false, false)
	if err != nil {
		errorutil.LogFatalf(err, "could not init content")
	}

	pub := pub{}

	logreader := logsource.NewReader(logSource, &pub)

	if err := logreader.Run(); err != nil {
		errorutil.LogFatalf(err, "import only")
	}

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
