// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package escalator

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/detective"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

type Request struct {
	Sender    string                `json:"sender"`
	Recipient string                `json:"recipient"`
	Interval  timeutil.TimeInterval `json:"time_interval"`
}

type Stepper interface {
	Step(func(Request) error, func() error) error
}

type Creator interface {
	Create(ctx context.Context, from, to string, interval timeutil.TimeInterval) error
}

type Requester interface {
	Request(Request)
}

type Escalator interface {
	Stepper
	Requester
}

type escalator struct {
	requests chan Request
}

func (e *escalator) Step(withResults func(Request) error, withoutResults func() error) error {
	select {
	case r := <-e.requests:
		return withResults(r)
	default:
		return withoutResults()
	}
}

func (e *escalator) Request(r Request) {
	e.requests <- r
}

func TryToEscalateRequest(ctx context.Context, detective detective.Detective, requester Requester, from, to string, interval timeutil.TimeInterval) error {
	messages, err := detective.CheckMessageDelivery(ctx, from, to, interval)
	if err != nil {
		return errorutil.Wrap(err)
	}

	shouldRefuse := func() bool {
		for _, m := range messages {
			// accept request only if at least one of the results came positive
			if m.Status == "bounced" || m.Status == "deferred" {
				return false
			}
		}

		return true
	}()

	if shouldRefuse {
		return nil
	}

	requester.Request(Request{
		Sender:    from,
		Recipient: to,
		Interval:  interval,
	})

	return nil
}

func New() Escalator {
	return &escalator{
		requests: make(chan Request),
	}
}
