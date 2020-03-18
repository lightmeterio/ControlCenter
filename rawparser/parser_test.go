package rawparser

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestParsingInvalidLines(t *testing.T) {
	Convey("Invalid Line", t, func() {
		parsed, err := ParseLogLine([]byte("Invalid Line"))
		So(parsed, ShouldEqual, nil)
		So(err, ShouldEqual, InvalidHeaderLineError)
	})
}

func TestParsingUnsupportedGeneralMessage(t *testing.T) {
	Convey("Unsupported Smtp Line", t, func() {
		parsed, err := ParseLogLine([]byte(`Sep 16 00:07:41 smtpnode07 postfix-10.20.30.40/smtp[31868]: 0D59F4165A:` +
			` host mx-aol.mail.gm0.yahoodns.net[44.55.66.77.88] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1;i ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command)`))
		So(parsed, ShouldEqual, nil)
		So(err, ShouldEqual, UnsupportedLogLineError)
	})

	Convey("Unsupported Log Line", t, func() {
		parsed, err := ParseLogLine([]byte(`Sep 16 00:07:34 smtpnode07 postfix-10.20.30.40/qmgr[2342]: ` +
			`3A1973E542: from=<redacted@phplist.com>, size=11737, nrcpt=1 (queue active)`))
		So(parsed, ShouldEqual, nil)
		So(err, ShouldEqual, UnsupportedLogLineError)
	})
}

func TestSMTPParsing(t *testing.T) {
	Convey("Basic SMTP Status", t, func() {
		parsed, err := ParseLogLine([]byte(`Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
			`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
			`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred ` +
			`(host mx-aol.mail.gm0.yahoodns.net[11.22.33.44] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1; ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))`))
		So(parsed, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		p, cast := parsed.Payload.(RawSmtpSentStatus)
		So(cast, ShouldEqual, true)
		So(p, ShouldNotEqual, nil)

		So(string(parsed.Header.Time), ShouldEqual, "Sep 16 00:07:43")
		So(string(parsed.Header.Month), ShouldEqual, "Sep")
		So(string(parsed.Header.Day), ShouldEqual, "16")
		So(string(parsed.Header.Hour), ShouldEqual, "00")
		So(string(parsed.Header.Minute), ShouldEqual, "07")
		So(string(parsed.Header.Second), ShouldEqual, "43")
		So(string(parsed.Header.Host), ShouldEqual, "smtpnode07")
		So(string(parsed.Header.Process), ShouldEqual, "smtp")
		So(string(p.Queue), ShouldEqual, "0C31D3D1E6")
		So(string(p.RecipientLocalPart), ShouldEqual, "redacted")
		So(string(p.RecipientDomainPart), ShouldEqual, "aol.com")
		So(string(p.RelayName), ShouldEqual, "mx-aol.mail.gm0.yahoodns.net")
		So(string(p.RelayIp), ShouldEqual, "11.22.33.44")
		So(string(p.RelayPort), ShouldEqual, "25")
		So(string(p.Delay), ShouldEqual, "18910")
		So(string(p.Delays[0]), ShouldEqual, "18900/8.9/0.69/0.03")
		So(string(p.Delays[1]), ShouldEqual, "18900")
		So(string(p.Delays[2]), ShouldEqual, "8.9")
		So(string(p.Delays[3]), ShouldEqual, "0.69")
		So(string(p.Delays[4]), ShouldEqual, "0.03")
		So(string(p.Dsn), ShouldEqual, "4.7.0")
		So(string(p.Status), ShouldEqual, "deferred")
		So(string(p.ExtraMessage), ShouldEqual, `(host mx-aol.mail.gm0.yahoodns.net[11.22.33.44] said: 421 4.7.0 [TSS04] `+
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1; `+
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))`)
	})
}
