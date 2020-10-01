package localrblinsight

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/localrbl"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"net"
	"time"
)

type Options struct {
	Checker       localrbl.Checker
	CheckInterval time.Duration
}

type content struct {
	ScanInterval data.TimeInterval         `json:"scan_interval"`
	Address      net.IP                    `json:"address"`
	RBLs         []localrbl.ContentElement `json:"rbls"`
}

func (c content) String() string {
	return fmt.Sprintf("The local IP %v has been blocked by %d RBLs", c.Address, len(c.RBLs))
}

const ContentType = "local_rbl_check"

func init() {
	core.RegisterContentType(ContentType, 4, func(b []byte) (core.Content, error) {
		content := content{}
		err := json.Unmarshal(b, &content)

		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		return &content, nil
	})
}

type detector struct {
	options Options
	creator core.Creator
}

func (d *detector) Close() error {
	return d.options.Checker.Close()
}

func getDetectorOptions(options core.Options) Options {
	detectorOptions, ok := options["localrbl"].(Options)

	if !ok {
		errorutil.MustSucceed(errors.New("Invalid detector options"), "")
	}

	return detectorOptions
}

func NewDetector(creator core.Creator, options core.Options) core.Detector {
	detectorOptions := getDetectorOptions(options)

	return &detector{
		options: detectorOptions,
		creator: creator,
	}
}

func createInsightForResults(d *detector, r localrbl.Results, c core.Clock, tx *sql.Tx) error {
	return generateInsight(tx, c, d.creator, content{
		ScanInterval: r.Interval,
		Address:      d.options.Checker.CheckedIP(),
		RBLs:         r.RBLs,
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

	d.options.Checker.NotifyNewScan(now)

	if err := core.StoreLastDetectorExecution(tx, "local_rbl_scan_start", now); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (d *detector) Step(c core.Clock, tx *sql.Tx) error {
	return d.options.Checker.Step(c.Now(), func(r localrbl.Results) error {
		// a scan result is available
		return createInsightForResults(d, r, c, tx)
	}, func() error {
		// no scan result available
		return maybeStartANewScan(d, c, tx)
	})
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
		Address:      d.options.Checker.CheckedIP(),
		RBLs: []localrbl.ContentElement{
			{RBL: "rbl.com", Text: "Funny reason"},
			{RBL: "anotherrbl.de", Text: "Another funny reason"},
		},
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
