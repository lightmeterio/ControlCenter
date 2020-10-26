package messagerbl

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	parser "gitlab.com/lightmeter/postfix-log-parser"
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

func pattern(p string) *regexp.Regexp {
	return regexp.MustCompile(p)
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
}

func newDetectorWithMatchers(settings globalsettings.Getter, matchers matchers) *Detector {
	return &Detector{
		Getter:           settings,
		nonDeliveredChan: make(chan record, 1024),
		resultsChan:      make(chan Result, 1024),
		matchers:         matchers,
	}
}

func (d *Detector) NewPublisher() *Publisher {
	return &Publisher{nonDeliveredChan: d.nonDeliveredChan}
}

func messageMatchesAnyHosts(settings globalsettings.Getter, matchers matchers, r record) (Result, bool) {
	for _, m := range matchers {
		if m.match(r) {
			return Result{
				Address: settings.IPAddress(context.Background()),
				Host:    m.host,
				Header:  r.header,
				Payload: r.payload,
				Time:    r.time,
			}, true
		}
	}

	return Result{}, false
}

func (d *Detector) Run() (done func(), cancel func()) {
	// Receives messages, filters them, and notify any listeners of if
	cancelChan := make(chan struct{})
	doneChan := make(chan struct{})

	go func() {
		<-cancelChan
		close(d.nonDeliveredChan)
	}()

	go func() {
		for r := range d.nonDeliveredChan {
			if result, matched := messageMatchesAnyHosts(d, d.matchers, r); matched {
				d.resultsChan <- result
			}
		}

		close(d.resultsChan)
		doneChan <- struct{}{}
	}()

	return func() { <-doneChan }, func() { cancelChan <- struct{}{} }
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
