package localrbl

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/mrichman/godnsbl"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"log"
	"net"
	"sync"
	"time"
)

var (
	DefaultRBLs = godnsbl.Blacklists
)

type contentElem struct {
	RBL  string `json:"rbl"`
	Text string `json:"text"`
}

type content struct {
	Address string        `json:"address"`
	RBLs    []contentElem `json:"rbls"`
}

const ContentType = "local_rbl_check"

type Options struct {
	CheckedAddress   net.IP
	CheckInterval    time.Duration
	RBLProvidersURLs []string
}

type checkResults struct {
	rbls []contentElem
}

type checker struct {
	checkerStartChan   chan struct{}
	checkerResultsChan chan checkResults
	options            Options
}

func startNewScan(checker *checker) {
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

func spawnChecker(checker *checker) {
	for range checker.checkerStartChan {
		log.Println("RBL Checker goroutine received notification for a new scan!")
		go startNewScan(checker)
	}

	log.Println("RBL Checker goroutine asked to stop!")
}

type detector struct {
	options Options
	creator core.Creator

	checker *checker
}

func (d *detector) Close() error {
	close(d.checker.checkerStartChan)
	return nil
}

func NewDetector(creator core.Creator, options core.Options) *detector {
	detectorOptions, ok := options["localrbl"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options!"), "")
	}

	checker := &checker{
		checkerStartChan:   make(chan struct{}, 32),
		checkerResultsChan: make(chan checkResults),
		options:            detectorOptions,
	}

	go spawnChecker(checker)

	return &detector{
		options: detectorOptions,
		creator: creator,
		checker: checker,
	}
}

func createInsightForResults(d *detector, r checkResults, c core.Clock, tx *sql.Tx) error {
	log.Println("Creating a new RBL Insight!")

	return generateInsight(tx, c, d.creator, content{
		Address: d.options.CheckedAddress.String(),
		RBLs:    r.rbls,
	})
}

func maybeStartANewScan(d *detector, c core.Clock, tx *sql.Tx) error {
	now := c.Now()

	// If it's time, ask the checker to start a new scan
	t, err := core.RetrieveLastDetectorExecution(tx, "local_rbl_scan_start")

	if err != nil {
		return errorutil.Wrap(err)
	}

	if !t.IsZero() && !now.After(t.Add(d.options.CheckInterval)) {
		return nil
	}

	// signal a new scan
	log.Println("Started notifying a new scan from the insights main loop!")
	d.checker.checkerStartChan <- struct{}{}
	log.Println("Finished notifying a new scan from the insights main loop!")

	if err := core.StoreLastDetectorExecution(tx, "local_rbl_scan_start", now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	select {
	case r := <-d.checker.checkerResultsChan:
		return createInsightForResults(d, r, c, tx)
	default:
		return maybeStartANewScan(d, c, tx)
	}
}

func (d *detector) Steppers() []core.Stepper {
	return []core.Stepper{d}
}

func generateInsight(tx *sql.Tx, c core.Clock, creator core.Creator, content content) error {
	properties := core.InsightProperties{
		Time:        c.Now(),
		Category:    core.LocalCategory,
		Rating:      core.BadRating,
		ContentType: ContentType,
		Content:     content,
	}

	if err := creator.GenerateInsight(tx, properties); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, content{
		Address: d.options.CheckedAddress.String(),
		RBLs: []contentElem{
			{RBL: "rbl.com", Text: "Funny reason"},
			{RBL: "anotherrbl.de", Text: "Another funny reason"},
		},
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func init() {
	core.RegisterContentType(ContentType, 4, func(b []byte) (interface{}, error) {
		content := content{}
		err := json.Unmarshal(b, &content)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &content, nil
	})
}
