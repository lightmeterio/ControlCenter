package localrbl

import (
	"context"
	"github.com/mrichman/godnsbl"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/meta"
	"log"
	"net"
	"sync"
	"time"
)

const (
	SettingsKey = "localrbl"
)

type Settings struct {
	LocalIP net.IP `json:"local_ip"`
}

type DNSLookupFunction func(string, string) godnsbl.RBLResults

var (
	RealLookup DNSLookupFunction = godnsbl.Lookup
)

type dnsChecker struct {
	checkerStartChan   chan time.Time
	checkerResultsChan chan Results
	options            Options
	meta               *meta.Reader
}

func newDnsChecker(meta *meta.Reader, options Options) *dnsChecker {
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

func NewChecker(meta *meta.Reader, options Options) Checker {
	return newDnsChecker(meta, options)
}

func (c *dnsChecker) CheckedIP(ctx context.Context) net.IP {
	var settings Settings
	err := c.meta.RetrieveJson(ctx, SettingsKey, &settings)

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

func startNewScan(checker *dnsChecker, t time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)

	defer cancel()

	results := make([]godnsbl.Result, len(checker.options.RBLProvidersURLs))

	ip := checker.CheckedIP(ctx)

	if err := ctx.Err(); err != nil {
		log.Println("Error obtaining IP address from settings on RBL Check:", err)
		return
	}

	if ip == nil {
		// Do not perform a scan if the user has not configured an IP
		return
	}

	log.Println("Starting a new RBL scan on IP", ip)

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

	log.Println("RBL scan finished in", scanElapsedTime)

	rbls := make([]ContentElement, 0, numberOfURLs)

	for _, r := range results {
		if r.Listed {
			rbls = append(rbls, ContentElement{RBL: r.Rbl, Text: r.Text})
		}
	}

	if len(rbls) == 0 {
		return
	}

	log.Println("RBL scan finished with", len(rbls), "list blockings")

	checker.checkerResultsChan <- Results{
		Interval: data.TimeInterval{From: t, To: t.Add(scanElapsedTime)},
		RBLs:     rbls,
	}
}