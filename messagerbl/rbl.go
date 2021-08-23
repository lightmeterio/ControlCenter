// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package messagerbl

import (
	"context"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/pkg/runner"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"net"
	"regexp"
	"time"
)

type record struct {
	time    time.Time
	header  parser.Header
	payload parser.SmtpSentStatus
}

type Publisher struct {
	nonDeliveredChan chan<- record
}

type matcher struct {
	host    string
	dsn     string
	pattern *regexp.Regexp
}

func (m matcher) match(r record) bool {
	// If dsn code is not available, ignore it
	dsnMatches := (len(m.dsn) > 0 && r.payload.Dsn == m.dsn) || len(m.dsn) == 0
	matches := dsnMatches && m.pattern.Match([]byte(r.payload.ExtraMessage))

	return matches
}

type matchers []matcher

func (p *Publisher) Publish(r postfix.Record) {
	// NOTE: We do the filtering here as writing to the channel is potentially blocking,
	// and this `if` is deterministic in behaviour, whereas the proper filtering
	// will happen in the other side of the channel is not and can block.
	if s, ok := r.Payload.(parser.SmtpSentStatus); ok && s.Status != parser.SentStatus {
		p.nonDeliveredChan <- record{payload: s, time: r.Time, header: r.Header}
	}
}

type Result struct {
	Address net.IP
	Host    string
	Payload parser.SmtpSentStatus
	Header  parser.Header
	Time    time.Time
}

type Detector struct {
	nonDeliveredChan chan record
	resultsChan      chan Results
	matchers         matchers
	runner.CancellableRunner
}

const (
	// MsgBufferSize is how much messages we are able to process without blocking
	// any other publisher.
	// Its value is for now arbitrary, and chosen experimentally to make the log thread
	// never block (see the deliverydb package)
	MsgBufferSize = 1024
)

func New() *Detector {
	d := &Detector{
		nonDeliveredChan: make(chan record, MsgBufferSize),
		resultsChan:      make(chan Results, MsgBufferSize),
		matchers:         defaultMatchers,
	}

	execute := func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			close(d.nonDeliveredChan)
		}()

		go func() {
			results := Results{}

			tryToFlush := func() {
				if results.Size > 0 {
					log.Debug().Msgf("Flushing %v messages", results.Size)
					d.resultsChan <- results
					results.Size = 0
				}
			}

			ticker := time.NewTicker(500 * time.Millisecond)

			// accumulate results in a buffer and notifies them when the buffer is full, or at timeout
			for {
				select {
				case r, ok := <-d.nonDeliveredChan:
					if !ok {
						log.Debug().Msg("Finished. Will just flush and leave")

						tryToFlush()

						close(d.resultsChan)
						done <- nil

						return
					}

					result, matched := messageMatchesAnyHosts(d.matchers, r)

					if !matched {
						break
					}

					results.Values[results.Size] = result
					results.Size++

					if results.Size == ResultsCapacity {
						tryToFlush()
					}
				case <-ticker.C:
					tryToFlush()
				}
			}
		}()
	}

	d.CancellableRunner = runner.NewCancellableRunner(execute)

	return d
}

func (d *Detector) NewPublisher() *Publisher {
	return &Publisher{nonDeliveredChan: d.nonDeliveredChan}
}

// TODO: this function should be optimized to lookup all patterns in a single shot.
// One way to implement it is to convert all individual regexes into a single huge
// automata. Ragel is a good candidate for it.
func messageMatchesAnyHosts(matchers matchers, r record) (Result, bool) {
	for _, m := range matchers {
		if m.match(r) {
			ip := func() net.IP {
				if r.header.ProcessIP != nil {
					return r.header.ProcessIP
				}

				return globalsettings.IPAddress(context.Background())
			}()

			return Result{
				Address: ip,
				Host:    m.host,
				Header:  r.header,
				Payload: r.payload,
				Time:    r.time,
			}, true
		}
	}

	return Result{}, false
}

const ResultsCapacity = 128

type Results struct {
	Values [ResultsCapacity]Result
	Size   int
}

type Stepper interface {
	Step(withResult func([]Results) error) error
}

func (d *Detector) Step(withResults func([]Results) error) error {
	var allResults []Results = nil

	for {
		select {
		case results, ok := <-d.resultsChan:
			// when the application ends, the channel closes and we need to leave
			if !ok {
				return withResults(allResults)
			}

			allResults = append(allResults, results)
		default:
			return withResults(allResults)
		}
	}
}
