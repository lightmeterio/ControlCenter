package meta

import (
	"context"
)

// Runner aims to serialize all requests to write in a single goroutine,
// which effectively owns writing access to the connection
type Runner struct {
	writer       *Writer
	requestsChan chan storeRequest
}

func NewRunner(h *Handler) *Runner {
	return &Runner{writer: h.Writer, requestsChan: make(chan storeRequest, 64)}
}

// AsyncWriter allows callers to schedule values to be stored, but in a non-blocking way
type AsyncWriter struct {
	runner *Runner
}

type storeRequest struct {
	items     []Item
	jsonKey   interface{}
	jsonValue interface{}
	errChan   chan<- error
}

type AsyncWriteResult struct {
	errChan <-chan error
}

func (r *AsyncWriteResult) Done() <-chan error {
	return r.errChan
}

// Wait forces the caller to wait until the underlying store call ends,
// either successfully or not
func (r *AsyncWriteResult) Wait() error {
	return <-r.Done()
}

func (runner *Runner) Writer() *AsyncWriter {
	return &AsyncWriter{runner: runner}
}

func (w *AsyncWriter) Store(items []Item) *AsyncWriteResult {
	c := make(chan error, 1)

	w.runner.requestsChan <- storeRequest{items: items, errChan: c}

	return &AsyncWriteResult{errChan: c}
}

func (w *AsyncWriter) StoreJson(key, value interface{}) *AsyncWriteResult {
	c := make(chan error, 1)

	w.runner.requestsChan <- storeRequest{jsonKey: key, jsonValue: value, errChan: c}

	return &AsyncWriteResult{errChan: c}
}

func runnerLoop(writer *Writer, requestsChan chan storeRequest) {
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
