package localrbl

import (
	"github.com/mrichman/godnsbl"
	"gitlab.com/lightmeter/controlcenter/data"
	"log"
	"sync"
	"time"
)

var (
	DefaultRBLs = godnsbl.Blacklists
)

var (
	defaultLookup = godnsbl.Lookup
)

type dnsChecker struct {
	checkerStartChan   chan time.Time
	checkerResultsChan chan checkResults
	options            Options
	lookup             func(string, string) godnsbl.RBLResults
}

func newDnsChecker(lookup func(string, string) godnsbl.RBLResults, options Options) *dnsChecker {
	return &dnsChecker{
		checkerStartChan:   make(chan time.Time, 32),
		checkerResultsChan: make(chan checkResults),
		options:            options,
		lookup:             lookup,
	}
}

func (c *dnsChecker) Close() error {
	close(c.checkerStartChan)
	return nil
}

func (c *dnsChecker) startListening() {
	go spawnChecker(c)
}

func (c *dnsChecker) notifyNewScan(t time.Time) {
	// signal a new scan
	log.Println("Started notifying a new scan from the insights main loop!")
	c.checkerStartChan <- t
	log.Println("Finished notifying a new scan from the insights main loop!")
}

func (c *dnsChecker) step(_ time.Time, withResults func(checkResults) error, withoutResults func() error) error {
	select {
	case r := <-c.checkerResultsChan:
		return withResults(r)
	default:
		return withoutResults()
	}
}

func startNewScan(checker *dnsChecker, t time.Time) {
	wg := &sync.WaitGroup{}

	type queryResult = godnsbl.Result

	results := make([]queryResult, len(checker.options.RBLProvidersURLs))

	ip := checker.options.CheckedAddress.String()

	log.Println("Starting a new RBL scan on IP", ip)

	scanStartTime := time.Now()

	for i, rbl := range checker.options.RBLProvidersURLs {
		wg.Add(1)

		go func(i int, rbl string) {
			defer wg.Done()

			r := checker.lookup(rbl, ip)

			if len(r.Results) > 0 {
				results[i] = r.Results[0]
			}
		}(i, rbl)
	}

	wg.Wait()

	scanEndTime := time.Now()

	rbls := make([]contentElem, 0, len(checker.options.RBLProvidersURLs))

	for _, r := range results {
		if r.Listed {
			rbls = append(rbls, contentElem{RBL: r.Rbl, Text: r.Text})
		}
	}

	if len(rbls) == 0 {
		log.Println("I am not in any lists!")
		return
	}

	log.Println("RBL scan finished with", len(rbls), "lists blocking me!")

	checker.checkerResultsChan <- checkResults{
		interval: data.TimeInterval{From: t, To: t.Add(scanEndTime.Sub(scanStartTime))},
		rbls:     rbls,
	}
}

func spawnChecker(checker *dnsChecker) {
	for t := range checker.checkerStartChan {
		log.Println("RBL Checker goroutine received notification for a new scan!")
		go startNewScan(checker, t)
	}

	log.Println("RBL Checker goroutine asked to stop!")
}
