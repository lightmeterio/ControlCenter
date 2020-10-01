package subcommand

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"strings"
	"testing"
	"time"
)

func TestOnlyImportLogs(t *testing.T) {
	Convey("Only Import Logs", t, func() {
		dir, clearDir := testutil.TempDir()
		defer clearDir()

		Convey("Read three lines", func() {
			reader := strings.NewReader(`

Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: 0C31D3D1E6: to=<a@b.c>, relay=a.net[1.2.3.4]:25, delay=1, delays=0/0.9/0.69/0.03, dsn=4.7.0, status=deferred Extra text)
Nov  1 07:42:10 mail opendkim[225]: C11EA2C620C7: not authenticated
			`)

			OnlyImportLogs(dir, time.UTC, 2020, true, reader)
		})
	})
}
