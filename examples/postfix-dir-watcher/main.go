package main

// To profile directory importing with a nice UI, execute:
// ./postfix-dir-watcher -dir /path/to/logs/dir -cpuprofile cpu.out -memprofile mem.out
// go run github.com/google/pprof -http ":6061" mem.out // or cpu.out
// and open your browser on http://localhost:6061

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/logeater/dirwatcher"
)

type pub struct {
}

func (p *pub) Publish(r data.Record) {
	if r.Payload == nil {
		return
	}

	j, err := json.Marshal(map[string]interface{}{
		"header":  r.Header,
		"payload": r.Payload,
	})

	if err != nil {
		log.Fatalln("JSON Error:", err)
	}

	fmt.Println(string(j))
}

func (p *pub) Close() {
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
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	if len(*dirToWatch) == 0 {
		log.Fatalln("-dir is mandatory!")
	}

	content, err := dirwatcher.NewDirectoryContent(*dirToWatch)

	if err != nil {
		log.Fatalln("Error:", err)
	}

	initialTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

	pub := pub{}

	watcher := dirwatcher.NewDirectoryImporter(content, &pub, initialTime)

	if err := watcher.ImportOnly(); err != nil {
		log.Fatalln("Error: ", err)
	}

	// copied from https://golang.org/pkg/runtime/pprof/
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
