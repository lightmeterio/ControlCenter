// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package escalator

import (
	"context"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/detective"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
)

type Request struct {
	Sender    string                `json:"sender"`
	Recipient string                `json:"recipient"`
	Interval  timeutil.TimeInterval `json:"time_interval"`
	Messages  detective.Messages    `json:"messages"`
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

func TryToEscalateRequest(ctx context.Context, d detective.Detective, requester Requester, from, to string, interval timeutil.TimeInterval) error {
	// FIXME: this call is only checking the first page, which is obviously wrong.
	// It should instead browse through all, by iterating over all pages!
	page := 1

	messages, err := d.CheckMessageDelivery(ctx, from, to, interval, -1, page)
	if err != nil {
		return errorutil.Wrap(err)
	}

	messagesToEscalate := detective.Messages{}

	for _, groupedByQueue := range messages.Messages {
		// NOTE: len(groupedByQueue) is always > 0, otherwise we have a bug!!!
		// The final status is always the last element in the list, as before a `sent` or `expired`
		// there might be many `deferred`
		// accept request only if at least one of the results came positive
		m := groupedByQueue.Entries[len(groupedByQueue.Entries)-1]

		if parser.SmtpStatus(m.Status) != parser.SentStatus {
			messagesToEscalate = append(messagesToEscalate, groupedByQueue)
		}
	}

	if len(messagesToEscalate) == 0 {
		log.Info().Msgf("Refused to escalate issue with sender: %v, recipient: %v and time interval: %v and %v results", from, to, interval, len(messages.Messages))
		return nil
	}

	log.Info().Msgf("Escalating issue with sender: %v, recipient: %v and time interval: %v and %v results", from, to, interval, len(messages.Messages))

	requester.Request(Request{
		Sender:    from,
		Recipient: to,
		Interval:  interval,
		Messages:  messagesToEscalate,
	})

	return nil
}

func New() Escalator {
	return &escalator{
		requests: make(chan Request, 1024),
	}
}
