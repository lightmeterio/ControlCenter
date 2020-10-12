package meta

import (
	"context"
)

type Runner struct {
	writer       *Writer
	requestsChan chan setRequest
}

func NewRunner(h *Handler) *Runner {
	return &Runner{writer: h.Writer, requestsChan: make(chan setRequest, 64)}
}

type setRequest struct {
	items   []Item
	errChan chan<- error
}

type Result struct {
	errChan <-chan error
}

func (r *Result) Wait() error {
	return <-r.errChan
}

func (runner *Runner) Store(items []Item) *Result {
	c := make(chan error, 1)

	runner.requestsChan <- setRequest{items: items, errChan: c}

	return &Result{errChan: c}
}

func (runner *Runner) Run() (done func(), cancel func()) {
	doneChan := make(chan struct{})
	cancelChan := make(chan struct{})

	go func() {
		<-cancelChan
		close(runner.requestsChan)
	}()

	go func() {
		ctx := context.Background()

		for req := range runner.requestsChan {
			err := runner.writer.Store(ctx, req.items)
			req.errChan <- err
			close(req.errChan)
		}

		doneChan <- struct{}{}
	}()

	return func() {
			<-doneChan
		}, func() {
			cancelChan <- struct{}{}
		}
}
