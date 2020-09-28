package localrbl

import (
	"github.com/mrichman/godnsbl"
	"log"
	"sync"
	"time"
)

var (
	DefaultRBLs = godnsbl.Blacklists
)

type dnsChecker struct {
	checkerStartChan   chan struct{}
	checkerResultsChan chan checkResults
	options            Options
}

func (c *dnsChecker) Close() error {
	close(c.checkerStartChan)
	return nil
}

func (c *dnsChecker) startListening() {
	go spawnChecker(c)
}

func (c *dnsChecker) notifyNewScan(time.Time) {
	// signal a new scan
	log.Println("Started notifying a new scan from the insights main loop!")
	c.checkerStartChan <- struct{}{}
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

func startNewScan(checker *dnsChecker) {
	wg := &sync.WaitGroup{}

	type queryResult = godnsbl.Result

	results := make([]queryResult, len(checker.options.RBLProvidersURLs))

	ip := checker.options.CheckedAddress.String()

	log.Println("Starting a new RBL scan on IP", ip)

	for i, rbl := range checker.options.RBLProvidersURLs {
		wg.Add(1)

		go func(i int, rbl string) {
			defer wg.Done()

			r := godnsbl.Lookup(rbl, ip)

			if len(r.Results) > 0 {
				results[i] = r.Results[0]
			}
		}(i, rbl)
	}

	wg.Wait()

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
		rbls: rbls,
	}
}

func spawnChecker(checker *dnsChecker) {
	for range checker.checkerStartChan {
		log.Println("RBL Checker goroutine received notification for a new scan!")
		go startNewScan(checker)
	}

	log.Println("RBL Checker goroutine asked to stop!")
}
