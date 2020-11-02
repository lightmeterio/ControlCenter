package messagerbl

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/data"
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

func (p *Publisher) Publish(r data.Record) {
	// NOTE: We do the filtering here as writing to the channel is potentially blocking,
	// and this `if` is deterministic in behaviour, whereas the proper filtering
	// will happen in the other side of the channel is not and can block.
	if s, ok := r.Payload.(parser.SmtpSentStatus); ok && s.Status != parser.SentStatus {
		p.nonDeliveredChan <- record{payload: s, time: r.Time, header: r.Header}
	}
}

func (p *Publisher) Close() {
}

type Result struct {
	Address net.IP
	Host    string
	Payload parser.SmtpSentStatus
	Header  parser.Header
	Time    time.Time
}

type Detector struct {
	globalsettings.Getter

	nonDeliveredChan chan record
	resultsChan      chan Result
	matchers         matchers
	runner.CancelableRunner
}

const (
	// MsgBufferSize is how much messages we are able to process without blocking
	// any other publisher.
	// Its value is for now arbitrary, and chosen experimentally to make the log thread
	// never block (see the logdb package)
	MsgBufferSize = 1024
)

func New(settings globalsettings.Getter) *Detector {
	d := &Detector{
		Getter:           settings,
		nonDeliveredChan: make(chan record, MsgBufferSize),
		resultsChan:      make(chan Result, MsgBufferSize),
		matchers:         defaultMatchers,
	}

	execute := func(done runner.DoneChan, cancel runner.CancelChan) {
		go func() {
			<-cancel
			close(d.nonDeliveredChan)
		}()

		go func() {
			for r := range d.nonDeliveredChan {
				if result, matched := messageMatchesAnyHosts(d, d.matchers, r); matched {
					d.resultsChan <- result
				}
			}

			close(d.resultsChan)
			done <- struct{}{}
		}()
	}

	d.CancelableRunner = runner.NewCancelableRunner(execute)

	return d
}

func (d *Detector) NewPublisher() *Publisher {
	return &Publisher{nonDeliveredChan: d.nonDeliveredChan}
}

func messageMatchesAnyHosts(settings globalsettings.Getter, matchers matchers, r record) (Result, bool) {
	for _, m := range matchers {
		if m.match(r) {
			ip := func() net.IP {
				if r.header.ProcessIP != nil {
					return r.header.ProcessIP
				}

				return settings.IPAddress(context.Background())
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

type Stepper interface {
	globalsettings.Getter
	Step(withResult func(Result) error, withoutResult func() error) error
}

func (d *Detector) Step(withResult func(Result) error, withoutResult func() error) error {
	select {
	case r := <-d.resultsChan:
		return withResult(r)
	default:
		return withoutResult()
	}
}
