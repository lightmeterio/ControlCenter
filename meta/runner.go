package meta

import (
	"context"
)

// Runner aims to serialize all requests to write in a single goroutine,
// wich effectivelly owns writing access to the connection
type Runner struct {
	writer       *Writer
	requestsChan chan setRequest
}

func NewRunner(h *Handler) *Runner {
	return &Runner{writer: h.Writer, requestsChan: make(chan setRequest, 64)}
}

type setRequest struct {
	items     []Item
	jsonKey   interface{}
	jsonValue interface{}
	errChan   chan<- error
}

type Result struct {
	errChan <-chan error
}

// Wait forces the caller to wait until the underlying store call ends,
// either successfully or not
func (r *Result) Wait() error {
	return <-r.errChan
}

func (runner *Runner) Store(items []Item) *Result {
	c := make(chan error, 1)

	runner.requestsChan <- setRequest{items: items, errChan: c}

	return &Result{errChan: c}
}

func (runner *Runner) StoreJson(key, value interface{}) *Result {
	c := make(chan error, 1)

	runner.requestsChan <- setRequest{jsonKey: key, jsonValue: value, errChan: c}

	return &Result{errChan: c}
}

func runnerLoop(writer *Writer, requestsChan chan setRequest) {
	ctx := context.Background()

	for req := range requestsChan {
		err := func() error {
			if req.jsonKey != nil {
				return writer.StoreJson(ctx, req.jsonKey, req.jsonValue)
			}

			return writer.Store(ctx, req.items)
		}()

		req.errChan <- err
		close(req.errChan)
	}
}

func (runner *Runner) Run() (done func(), cancel func()) {
	doneChan := make(chan struct{})
	cancelChan := make(chan struct{})

	go func() {
		<-cancelChan
		close(runner.requestsChan)
	}()

	go func() {
		runnerLoop(runner.writer, runner.requestsChan)
		doneChan <- struct{}{}
	}()

	return func() { <-doneChan }, func() { cancelChan <- struct{}{} }
}
