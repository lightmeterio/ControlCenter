package logeater

import (
	"bufio"
	"io"
	"log"

	"errors"

	"github.com/hpcloud/tail"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/util"
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

var (
	FileIsNotRegular = errors.New("File is not regular")
)

func WatchFileCancelable(filename string,
	location tail.SeekInfo,
	publisher data.Publisher) (error, chan<- interface{}, <-chan error) {

	t, err := tail.TailFile(filename, tail.Config{
		Follow:    true,
		ReOpen:    false,
		Logger:    tail.DiscardingLogger,
		MustExist: true,
		Location:  &location,
	})

	if err != nil {
		return err, nil, nil
	}

	cancel := make(chan interface{})
	done := make(chan error)

	cleanup := make(chan interface{})

	go func() {
		<-cancel
		util.MustSucceed(t.Stop(), "Stopping file watcher")
		cleanup <- nil
	}()

	go func() {
		for line := range t.Lines {
			tryToParseAndPublish([]byte(line.Text), publisher)
		}

		<-cleanup
		t.Cleanup()

		done <- nil
	}()

	return nil, cancel, done
}

func WatchFile(filename string, location tail.SeekInfo, publisher data.Publisher) error {
	// it ends only when the file is deleted (I guess, as its behaviour is defined in the tail package)
	err, _, done := WatchFileCancelable(filename, location, publisher)

	if err != nil {
		return err
	}

	if err := <-done; err != nil {
		return err
	}

	return err
}

func tryToParseAndPublish(line []byte, publisher data.Publisher) {
	h, p, err := parser.Parse(line)

	if !parser.IsRecoverableError(err) {
		log.Printf("Invalid Postfix header: \"%s\"", string(line))
		return
	}

	publisher.Publish(data.Record{Header: h, Payload: p})
}
