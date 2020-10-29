// +build dev !release

package messagerblinsight

import (
	"context"
	"database/sql"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

// Executed only on development builds, for better developer experience
func (d *detector) GenerateSampleInsight(tx *sql.Tx, c core.Clock) error {
	if err := generateInsight(tx, c, d.creator, content{
		Address:   d.options.Detector.IPAddress(context.Background()),
		Message:   "Sample Insight: host blocked. Try https://google.com/ to unblock it",
		Recipient: "some.mail.com",
		Status:    parser.DeferredStatus.String(),
		Host:      "Big Host",
		Time:      c.Now(),
	}); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
