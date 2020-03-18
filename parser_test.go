package parser

import (
	"encoding/hex"
	. "github.com/smartystreets/goconvey/convey"
	. "gitlab.com/lightmeter/postfix-logs-parser/rawparser"
	"net"
	"testing"
)

func TestParsingInvalidLines(t *testing.T) {
	Convey("Invalid Line", t, func() {
		_, err := Parse([]byte("Invalid Line"))
		So(err, ShouldEqual, InvalidHeaderLineError)
	})
}

func TestParsingUnsupportedGeneralMessage(t *testing.T) {
	Convey("Unsupported Smtp Line", t, func() {
		_, err := Parse([]byte(`Sep 16 00:07:41 smtpnode07 postfix-10.20.30.40/smtp[31868]: 0D59F4165A:` +
			` host mx-aol.mail.gm0.yahoodns.net[44.55.66.77.88] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1;i ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command)`))
		So(err, ShouldEqual, UnsupportedLogLineError)
	})

	Convey("Unsupported Log Line", t, func() {
		_, err := Parse([]byte(`Sep 16 00:07:34 smtpnode07 postfix-10.20.30.40/qmgr[2342]: ` +
			`3A1973E542: from=<redacted@phplist.com>, size=11737, nrcpt=1 (queue active)`))
		So(err, ShouldEqual, UnsupportedLogLineError)
	})
}

func TestSMTPParsing(t *testing.T) {
	Convey("Basic SMTP Status", t, func() {
		parsed, err := Parse([]byte(`Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
			`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
			`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred ` +
			`(host mx-aol.mail.gm0.yahoodns.net[11.22.33.44] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1; ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))`))
		So(parsed, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		p, cast := parsed.Payload.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(parsed.Header.Time.Day, ShouldEqual, 16)
		So(parsed.Header.Time.Month.String(), ShouldEqual, "September")
		So(parsed.Header.Time.Hour, ShouldEqual, 0)
		So(parsed.Header.Time.Minute, ShouldEqual, 7)
		So(parsed.Header.Time.Second, ShouldEqual, 43)
		So(string(parsed.Header.Host), ShouldEqual, "smtpnode07")
		So(parsed.Header.Process, ShouldEqual, SmtpProcess)

		q, _ := hex.DecodeString("0C31D3D1E6")

		So(string(p.Queue), ShouldEqual, string(q))
		So(p.RecipientLocalPart, ShouldEqual, "redacted")
		So(p.RecipientDomainPart, ShouldEqual, "aol.com")
		So(p.RelayName, ShouldEqual, "mx-aol.mail.gm0.yahoodns.net")

		So(p.RelayIP, ShouldEqual, net.ParseIP("11.22.33.44"))
		So(p.RelayPort, ShouldEqual, 25)
		So(p.Delay, ShouldEqual, 18910)
		So(p.Delays.Smtpd, ShouldEqual, 18900)
		So(p.Delays.Cleanup, ShouldEqual, 8.9)
		So(p.Delays.Qmgr, ShouldEqual, 0.69)
		So(p.Delays.Smtp, ShouldEqual, 0.03)
		So(p.Dsn, ShouldEqual, "4.7.0")
		So(p.Status, ShouldEqual, DeferredStatus)
		So(p.ExtraMessage, ShouldEqual, `(host mx-aol.mail.gm0.yahoodns.net[11.22.33.44] said: 421 4.7.0 [TSS04] `+
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1; `+
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))`)
	})
}
