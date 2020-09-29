package localrbl

import (
	"github.com/mrichman/godnsbl"
	"gitlab.com/lightmeter/controlcenter/data"
	"log"
	"net"
	"time"
)

var (
	DefaultRBLs = godnsbl.Blacklists
)

type DNSLookupFunction func(string, string) godnsbl.RBLResults

var (
	RealLookup DNSLookupFunction = godnsbl.Lookup
)

type dnsChecker struct {
	checkerStartChan   chan time.Time
	checkerResultsChan chan Results
	options            Options
}

func newDnsChecker(options Options) *dnsChecker {
	if options.NumberOfWorkers < 1 {
		log.Panicln("DnsChecker should have a number of workers greater than 1 and not", options.NumberOfWorkers, "!")
	}

	return &dnsChecker{
		checkerStartChan:   make(chan time.Time, 32),
		checkerResultsChan: make(chan Results),
		options:            options,
	}
}

func NewChecker(options Options) Checker {
	return newDnsChecker(options)
}

func (c *dnsChecker) CheckedIP() net.IP {
	// TODO: obtain such value from the application settings!
	return c.options.CheckedAddress
}

func (c *dnsChecker) Close() error {
	close(c.checkerStartChan)
	return nil
}

func (c *dnsChecker) StartListening() {
	go spawnChecker(c)
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
func worker(jobs <-chan func(), results chan<- struct{}) {
	for job := range jobs {
		job()
		results <- struct{}{}
	}
}

func startNewScan(checker *dnsChecker, t time.Time) {
	type queryResult = godnsbl.Result

	results := make([]queryResult, len(checker.options.RBLProvidersURLs))

	ip := checker.options.CheckedAddress.String()

	log.Println("Starting a new RBL scan on IP", ip)

	scanStartTime := time.Now()

	jobsChan := make(chan func(), len(checker.options.RBLProvidersURLs))
	resultsChan := make(chan struct{}, len(checker.options.RBLProvidersURLs))

	for w := 0; w < checker.options.NumberOfWorkers; w++ {
		go worker(jobsChan, resultsChan)
	}

	for i, rbl := range checker.options.RBLProvidersURLs {
		jobsChan <- func(i int, rbl string) func() {
			return func() {
				r := checker.options.Lookup(rbl, ip)

				if len(r.Results) > 0 {
					results[i] = r.Results[0]
				}
			}
		}(i, rbl)
	}

	close(jobsChan)

	for range checker.options.RBLProvidersURLs {
		<-resultsChan
	}

	scanEndTime := time.Now()

	rbls := make([]ContentElement, 0, len(checker.options.RBLProvidersURLs))

	for _, r := range results {
		if r.Listed {
			rbls = append(rbls, ContentElement{RBL: r.Rbl, Text: r.Text})
		}
	}

	if len(rbls) == 0 {
		log.Println("I am not in any lists!")
		return
	}

	log.Println("RBL scan finished with", len(rbls), "lists blocking me!")

	checker.checkerResultsChan <- Results{
		Interval: data.TimeInterval{From: t, To: t.Add(scanEndTime.Sub(scanStartTime))},
		RBLs:     rbls,
	}
}

func spawnChecker(checker *dnsChecker) {
	for t := range checker.checkerStartChan {
		go startNewScan(checker, t)
	}
}
