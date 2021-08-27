// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package meta

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// Runner aims to serialize all requests to write in a single goroutine,
// which effectively owns writing access to the connection
type Runner struct {
	db           *dbconn.DB
	requestsChan chan storeRequest
	runner.CancellableRunner
}

func NewRunner(db *dbconn.DB) *Runner {
	r := &Runner{db: db, requestsChan: make(chan storeRequest)}

	cancelableRunner := runner.NewCancellableRunner(func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			close(r.requestsChan)
		}()

		go func() {
			r.loop()
			done <- nil
		}()
	})

	r.CancellableRunner = cancelableRunner

	return r
}

// AsyncWriter allows callers to schedule values to be stored, but in a non-blocking way
// TODO: at the moment AsyncWriter serializes and doesn't bufferize the store requests,
// making them behave as if they were blocking for all the requesters.
// It can be a problem in case of "high pressure" with many simultaneous requests,
// which can be a bit slow.
// In such scenarios, one possibility to be verified is to accumulate many requests in a single transaction
// as SQLite can be slow on storing multiple independent pieces of data, but is quite efficient
// when grouping them into a single transaction.
type AsyncWriter struct {
	runner *Runner
}

// A request to store something, done asynchronously
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

func (w *AsyncWriter) StoreJsonSync(ctx context.Context, key, value interface{}) error {
	result := w.StoreJson(key, value)

	select {
	case err := <-result.Done():
		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	case <-ctx.Done():
		return errorutil.Wrap(ctx.Err())
	}
}

func (runner *Runner) handleRequest(ctx context.Context, req storeRequest) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)

	defer cancel()

	err := func() error {
		if req.jsonKey != nil {
			return StoreJson(ctx, runner.db, req.jsonKey, req.jsonValue)
		}

		return Store(ctx, runner.db, req.items)
	}()

	req.errChan <- err
	close(req.errChan)
}

func (runner *Runner) loop() {
	loopContext := context.Background()

	for req := range runner.requestsChan {
		runner.handleRequest(loopContext, req)
	}
}
