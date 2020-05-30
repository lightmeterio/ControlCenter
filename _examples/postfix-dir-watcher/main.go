package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
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

	j, err := json.MarshalIndent(map[string]interface{}{
		"header":  r.Header,
		"payload": r.Payload,
	}, "", "  ")

	if err != nil {
		log.Fatalln("JSON Error:", err)
	}

	fmt.Println(string(j))
}

func (p *pub) Close() {
}

func main() {
	dirToWatch := flag.String("dir", "", "Directory to watch")

	flag.Parse()

	if len(*dirToWatch) == 0 {
		log.Fatalln("-dir is mandatory!")
	}

	content, err := dirwatcher.NewDirectoryContent(*dirToWatch)

	if err != nil {
		log.Fatalln("Error:", err)
	}

	initialTime := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

	pub := pub{}

	watcher := dirwatcher.NewDirectoryImporter(content, &pub, time.UTC, initialTime)

	if err := watcher.Run(); err != nil {
		log.Fatalln("Error: ", err)
	}
}
