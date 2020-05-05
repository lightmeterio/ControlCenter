package logeater

import (
	"bufio"
	"io"
	"log"

	"github.com/hpcloud/tail"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/workspace"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

func ReadFromReader(reader io.Reader, pub data.Publisher) {
	scanner := bufio.NewScanner(reader)

	for {
		if !scanner.Scan() {
			break
		}

		tryToParseAndPublish(scanner.Bytes(), pub)
	}
}

// TODO: hm... it feels this function should belong to data.Workspace!
func FindWatchingLocationForWorkspace(ws *workspace.Workspace) tail.SeekInfo {
	if !ws.HasLogs() {
		return tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekCurrent,
		}
	}

	return tail.SeekInfo{
		Offset: 0,
		Whence: io.SeekEnd,
	}
}

func WatchFile(filename string, location tail.SeekInfo, publisher data.Publisher) error {
	t, err := tail.TailFile(filename, tail.Config{
		Follow:   true,
		ReOpen:   true,
		Logger:   tail.DiscardingLogger,
		Location: &location,
	})

	if err != nil {
		return err
	}

	for line := range t.Lines {
		tryToParseAndPublish([]byte(line.Text), publisher)
	}

	return nil
}

func tryToParseAndPublish(line []byte, publisher data.Publisher) {
	h, p, err := parser.Parse(line)

	if !parser.IsRecoverableError(err) {
		log.Printf("Invalid Postfix header: \"%s\"", string(line))
		return
	}

	publisher.Publish(data.Record{Header: h, Payload: p})
}
