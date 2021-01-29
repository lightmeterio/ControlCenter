// +build dev !release

package newsfeed

import (
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"time"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, Content{
		Title:       "Sample Newsfeed insight!!!",
		Description: "A new threat has been discovered. Protect yourself. Or die trying...",
		Link:        "http://lightmeter.io/updates",
		Published:   time.Date(2020, time.February, 22, 1, 2, 3, 0, time.UTC),
		GUID:        "http://lightmeter.io/?p=1",
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
