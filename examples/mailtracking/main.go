package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/logeater/dirlogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/filelogsource"
	"gitlab.com/lightmeter/controlcenter/logeater/logsource"
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

	s := map[string]interface{}{}

	for i, v := range r {
		if v != nil {
			s[tracking.KeysToLabels[i]] = v
		}
	}

	j, err := json.Marshal(s)

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
			return dirlogsource.New(inputDirectory, time.Time{}, false)
		}

		f, err := os.Open(inputFile)
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		year := time.Now().Year()

		return filelogsource.New(f, time.Time{}, year)
	}()

	errorutil.MustSucceed(err)

	pub := publisher{}

	t, err := tracking.New(workspace, &pub)

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
