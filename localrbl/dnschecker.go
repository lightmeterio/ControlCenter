package localrbl

import (
	"github.com/mrichman/godnsbl"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/meta"
	"log"
	"net"
	"time"
)

const (
	SettingsKey = "localrbl"
)

type Settings struct {
	LocalIP net.IP `json:"local_ip"`
}

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
	meta               *meta.MetadataHandler
}

func newDnsChecker(meta *meta.MetadataHandler, options Options) *dnsChecker {
	if options.NumberOfWorkers < 1 {
		log.Panicln("DnsChecker should have a number of workers greater than 1 and not", options.NumberOfWorkers, "!")
	}

	if options.Lookup == nil {
		log.Panicln("Lookup function not defined!")
	}

	return &dnsChecker{
		checkerStartChan:   make(chan time.Time, 32),
		checkerResultsChan: make(chan Results),
		options:            options,
		meta:               meta,
	}
}

func NewChecker(meta *meta.MetadataHandler, options Options) Checker {
	return newDnsChecker(meta, options)
}

func (c *dnsChecker) CheckedIP() net.IP {
	var settings Settings
	err := c.meta.RetrieveJson(SettingsKey, &settings)

	if err != nil {
		// If we cannot obtain the ip address, just chicken out
		return nil
	}

	return settings.LocalIP
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

	ip := checker.CheckedIP()

	if ip == nil {
		// Do not perform a scan if the user has not configured an IP
		return
	}

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
				r := checker.options.Lookup(rbl, ip.String())

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
