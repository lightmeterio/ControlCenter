// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package localrbl

import (
	"context"
	"errors"
	"github.com/mrichman/godnsbl"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"sync"
	"time"
)

type DNSLookupFunction func(string, string) godnsbl.RBLResults

var (
	RealLookup DNSLookupFunction = godnsbl.Lookup
)

type dnsChecker struct {
	*globalsettings.MetaReaderGetter

	checkerStartChan   chan time.Time
	checkerResultsChan chan Results
	options            Options
	meta               *meta.Reader
}

func newDnsChecker(meta *meta.Reader, options Options) *dnsChecker {
	if options.NumberOfWorkers < 1 {
		log.Panic().Msgf("DnsChecker should have a number of workers greater than 1 and not %d!", options.NumberOfWorkers)
	}

	if options.Lookup == nil {
		log.Panic().Msg("Lookup function not defined!")
	}

	return &dnsChecker{
		MetaReaderGetter:   globalsettings.New(meta),
		checkerStartChan:   make(chan time.Time, 32),
		checkerResultsChan: make(chan Results),
		options:            options,
		meta:               meta,
	}
}

func NewChecker(meta *meta.Reader, options Options) Checker {
	return newDnsChecker(meta, options)
}

func (c *dnsChecker) Close() error {
	close(c.checkerStartChan)
	return nil
}

func (c *dnsChecker) StartListening() {
	go func() {
		for t := range c.checkerStartChan {
			go startNewScan(c, t)
		}
	}()
}

func (c *dnsChecker) NotifyNewScan(t time.Time) {
	c.checkerStartChan <- t
}

func (c *dnsChecker) Step(_ time.Time, withResults func(Results) error, withoutResults func() error) error {
	select {
	case r := <-c.checkerResultsChan:
		return withResults(r)
	default:
		return withoutResults()
	}
}

// Honestly, this is almost copy&paste of https://gobyexample.com/worker-pools
func worker(jobs <-chan func(), wg *sync.WaitGroup) {
	for job := range jobs {
		job()
		wg.Done()
	}
}

var (
	ErrIPNotConfigured = errors.New(`IP not configured`)
)

// TODO: refactor this function into smaller pieces, as it's quite trivial
func startNewScan(checker *dnsChecker, t time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)

	defer cancel()

	results := make([]godnsbl.Result, len(checker.options.RBLProvidersURLs))

	ip := checker.IPAddress(ctx)

	if err := ctx.Err(); err != nil {
		errorutil.LogErrorf(err, "obtaining IP address from settings on RBL Check")
		checker.checkerResultsChan <- Results{Err: errorutil.Wrap(err)}

		return
	}

	if ip == nil {
		// Do not perform a scan if the user has not configured an IP
		log.Warn().Msg("Ignoring RBL scan as IP is not configured")
		checker.checkerResultsChan <- Results{Err: ErrIPNotConfigured}

		return
	}

	log.Info().Msgf("Starting a new RBL scan on IP %v", ip)

	scanStartTime := time.Now()

	numberOfURLs := len(checker.options.RBLProvidersURLs)

	jobsChan := make(chan func(), numberOfURLs)

	wg := sync.WaitGroup{}
	wg.Add(numberOfURLs)

	for w := 0; w < checker.options.NumberOfWorkers; w++ {
		go worker(jobsChan, &wg)
	}

	for i, rbl := range checker.options.RBLProvidersURLs {
		jobsChan <- func(i int, rbl string) func() {
			return func() {
				r := checker.options.Lookup(rbl, ip.String())

				if len(r.Results) > 0 {
					results[i] = r.Results[0]
				}
			}
		}(i, rbl)
	}

	close(jobsChan)

	wg.Wait()

	scanEndTime := time.Now()

	scanElapsedTime := scanEndTime.Sub(scanStartTime)

	log.Info().Msgf("RBL scan finished in %v", scanElapsedTime)

	rbls := make([]ContentElement, 0, numberOfURLs)

	for _, r := range results {
		if r.Listed {
			rbls = append(rbls, ContentElement{RBL: r.Rbl, Text: r.Text})
		}
	}

	if len(rbls) == 0 {
		return
	}

	log.Info().Msgf("RBL scan finished with list blockings %d", len(rbls))

	checker.checkerResultsChan <- Results{
		Interval: timeutil.TimeInterval{From: t, To: t.Add(scanElapsedTime)},
		RBLs:     rbls,
	}
}
