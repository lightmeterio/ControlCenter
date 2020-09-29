package localrbl

import (
	"database/sql"
	"encoding/json"
	"errors"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"net"
	"time"
)

type contentElem struct {
	RBL  string `json:"rbl"`
	Text string `json:"text"`
}

type content struct {
	ScanInterval data.TimeInterval `json:"scan_interval"`
	Address      string            `json:"address"`
	RBLs         []contentElem     `json:"rbls"`
}

const ContentType = "local_rbl_check"

type Options struct {
	NumberOfWorkers  int
	CheckedAddress   net.IP
	CheckInterval    time.Duration
	RBLProvidersURLs []string
}

type checkResults struct {
	interval data.TimeInterval
	rbls     []contentElem
}

type detector struct {
	options Options
	creator core.Creator
	checker checker
}

func (d *detector) Close() error {
	return d.checker.Close()
}

type checker interface {
	io.Closer

	startListening()
	notifyNewScan(time.Time)
	step(time.Time, func(checkResults) error, func() error) error
}

func getDetectorOptions(options core.Options) Options {
	detectorOptions, ok := options["localrbl"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options!"), "")
	}

	return detectorOptions
}

func newDetector(creator core.Creator, options core.Options, checker checker) *detector {
	detectorOptions := getDetectorOptions(options)

	return &detector{
		options: detectorOptions,
		creator: creator,
		checker: checker,
	}
}

func NewDetector(creator core.Creator, options core.Options) *detector {
	checker := newDnsChecker(defaultLookup, getDetectorOptions(options))

	checker.startListening()

	return newDetector(creator, options, checker)
}

func createInsightForResults(d *detector, r checkResults, c core.Clock, tx *sql.Tx) error {
	return generateInsight(tx, c, d.creator, content{
		ScanInterval: r.interval,
		Address:      d.options.CheckedAddress.String(),
		RBLs:         r.rbls,
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

	d.checker.notifyNewScan(now)

	if err := core.StoreLastDetectorExecution(tx, "local_rbl_scan_start", now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	return d.checker.step(c.Now(), func(r checkResults) error {
		// a scan result is available
		return createInsightForResults(d, r, c, tx)
	}, func() error {
		// no scan result available
		return maybeStartANewScan(d, c, tx)
	})
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
		ScanInterval: data.TimeInterval{From: c.Now(), To: c.Now().Add(time.Second * 30)},
		Address:      d.options.CheckedAddress.String(),
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
