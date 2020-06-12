package logeater

import (
	"bufio"
	"io"
	"log"

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

func WatchFileCancelable(filename string,
	location tail.SeekInfo,
	publisher data.Publisher) (err error, stopWatching func(), waitForDone func()) {

	t, err := tail.TailFile(filename, tail.Config{
		Follow:    true,
		ReOpen:    false,
		Logger:    tail.DiscardingLogger,
		MustExist: true,
		Location:  &location,
		Poll:      false,
	})

	if err != nil {
		return util.WrapError(err), nil, nil
	}

	cancel := make(chan struct{}, 1)
	done := make(chan error)

	go func() {
	loop:
		for {
			select {
			case line, ok := <-t.Lines:
				if !ok {
					break loop
				}

				tryToParseAndPublish([]byte(line.Text), publisher)
				break

			case <-cancel:
				break loop
			}
		}

		util.MustSucceed(t.Stop(), "stopping watcher")

		done <- nil
	}()

	return nil, func() {
			cancel <- struct{}{}
		}, func() {
			<-done
		}
}

func WatchFile(filename string, location tail.SeekInfo, publisher data.Publisher) error {
	// it ends only when the file is deleted (I guess, as its behaviour is defined in the tail package)
	err, _, done := WatchFileCancelable(filename, location, publisher)

	if err != nil {
		return util.WrapError(err)
	}

	done()

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
