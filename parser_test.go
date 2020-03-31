package parser

import (
	. "github.com/smartystreets/goconvey/convey"
	r "gitlab.com/lightmeter/postfix-log-parser/rawparser"
	"net"
	"testing"
)

func TestParsingInvalidLines(t *testing.T) {
	Convey("Invalid Line", t, func() {
		_, p, err := Parse([]byte("Invalid Line"))
		So(p, ShouldEqual, nil)
		So(err, ShouldEqual, r.InvalidHeaderLineError)
	})
}

func TestParsingUnsupportedGeneralMessage(t *testing.T) {
	Convey("Unsupported Smtp Line", t, func() {
		h, p, err := Parse([]byte(`Sep 16 00:07:41 smtpnode07 postfix-10.20.30.40/smtp[31868]: 0D59F4165A:` +
			` host mx-aol.mail.gm0.yahoodns.net[44.55.66.77] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1;i ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command)`))

		So(p, ShouldEqual, nil)
		So(err, ShouldEqual, r.UnsupportedLogLineError)
		So(h.Process, ShouldEqual, "smtp")
		So(h.Time.Day, ShouldEqual, 16)
		So(h.Time.Month.String(), ShouldEqual, "September")
		So(h.Time.Hour, ShouldEqual, 0)
		So(h.Time.Minute, ShouldEqual, 7)
		So(h.Time.Second, ShouldEqual, 41)
		So(h.Host, ShouldEqual, "smtpnode07")
		So(h.Process, ShouldEqual, "smtp")
	})

	Convey("Unsupported Log Line", t, func() {
		_, p, err := Parse([]byte(`Sep 16 00:07:34 smtpnode07 postfix-10.20.30.40/qmgr[2342]: ` +
			`3A1973E542: from=<redacted@phplist.com>, size=11737, nrcpt=1 (queue active)`))
		So(p, ShouldEqual, nil)
		So(err, ShouldEqual, r.UnsupportedLogLineError)
	})

	Convey("Unsupported Log Line with slash on process", t, func() {
		_, p, err := Parse([]byte(`Feb  3 02:55:42 mail postfix/submission/smtpd[21543]: connect from unknown[11.22.33.44]`))
		So(p, ShouldEqual, nil)
		So(err, ShouldEqual, r.UnsupportedLogLineError)
	})

	Convey("Unsupported opendkim line", t, func() {
		_, p, err := Parse([]byte(`Feb  5 19:00:02 mail opendkim[195]: 407032C4FF6A: DKIM-Signature field added (s=mail, d=lightmeter.io)`))
		So(p, ShouldEqual, nil)
		So(err, ShouldEqual, r.UnsupportedLogLineError)
	})

	Convey("Unsupported dovecot line", t, func() {
		_, p, err := Parse([]byte(`Feb  5 18:56:52 mail dovecot: imap(laal@mail.io)<28358><CO3htpid9tRXDNo5>: Connection closed (IDLE running for 0.001 + waiting input for 28.914 secs, 2 B in + 10 B out, state=wait-input) in=703 out=12338 deleted=0 expunged=0 trashed=0 hdr_count=0 hdr_bytes=0 body_count=0 body_bytes=0`))
		So(p, ShouldEqual, nil)
		So(err, ShouldEqual, r.UnsupportedLogLineError)
	})
}

func TestSMTPParsing(t *testing.T) {
	Convey("Basic SMTP Status", t, func() {
		header, parsed, err := Parse([]byte(`Sep 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
			`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
			`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred ` +
			`(host mx-aol.mail.gm0.yahoodns.net[11.22.33.44] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1; ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))`))
		So(parsed, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 16)
		So(header.Time.Month.String(), ShouldEqual, "September")
		So(header.Time.Hour, ShouldEqual, 0)
		So(header.Time.Minute, ShouldEqual, 7)
		So(header.Time.Second, ShouldEqual, 43)
		So(header.Host, ShouldEqual, "smtpnode07")
		So(header.Process, ShouldEqual, "smtp")

		So(p.Queue, ShouldEqual, "0C31D3D1E6")
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

	Convey("Basic SMTP Status from different logs", t, func() {
		header, parsed, err := Parse([]byte(`Feb  5 17:24:35 mail postfix/smtp[9635]: D298F2C60812: to=<"user 1234 with space"@icloud.com>,` +
			` relay=mx6.mail.icloud.com[17.178.97.79]:25, delay=428621, delays=428619/0.02/1.9/0, ` +
			`dsn=4.7.0, status=deferred (host mx6.mail.icloud.com[17.178.97.79] ` +
			`refused to talk to me: 550 5.7.0 Blocked - see https://support.proofpoint.com/dnsbl-lookup.cgi?ip=142.93.169.220)`))
		So(parsed, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 5)
		So(header.Time.Month.String(), ShouldEqual, "February")
		So(header.Time.Hour, ShouldEqual, 17)
		So(header.Time.Minute, ShouldEqual, 24)
		So(header.Time.Second, ShouldEqual, 35)
		So(header.Host, ShouldEqual, "mail")
		So(header.Process, ShouldEqual, "smtp")

		So(p.Queue, ShouldEqual, "D298F2C60812")
		So(p.RecipientLocalPart, ShouldEqual, "user 1234 with space")
		So(p.RecipientDomainPart, ShouldEqual, "icloud.com")
		So(p.RelayName, ShouldEqual, "mx6.mail.icloud.com")

		So(p.RelayIP, ShouldEqual, net.ParseIP("17.178.97.79"))
		So(p.RelayPort, ShouldEqual, 25)
		So(p.Delay, ShouldEqual, 428621)
		So(p.Delays.Smtpd, ShouldEqual, 428619)
		So(p.Delays.Cleanup, ShouldEqual, 0.02)
		So(p.Delays.Qmgr, ShouldEqual, 1.9)
		So(p.Delays.Smtp, ShouldEqual, 0)
		So(p.Dsn, ShouldEqual, "4.7.0")
		So(p.Status, ShouldEqual, DeferredStatus)
	})

	Convey("A bounced message", t, func() {
		header, parsed, err := Parse([]byte(`Feb  3 04:41:17 mail postfix/smtp[10603]: AE8E32C60819: to=<mail@e.mail.com>, ` +
			`relay=none, delay=0.02, delays=0.01/0/0.01/0, dsn=5.4.4, status=bounced ` +
			`(Host or domain name not found. Name service error for name=e.mail.com type=AAAA: Host not found)`))
		So(parsed, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 3)
		So(header.Time.Month.String(), ShouldEqual, "February")
		So(header.Time.Hour, ShouldEqual, 4)
		So(header.Time.Minute, ShouldEqual, 41)
		So(header.Time.Second, ShouldEqual, 17)
		So(header.Host, ShouldEqual, "mail")
		So(header.Process, ShouldEqual, "smtp")

		So(p.Queue, ShouldEqual, "AE8E32C60819")
		So(p.RecipientLocalPart, ShouldEqual, "mail")
		So(p.RecipientDomainPart, ShouldEqual, "e.mail.com")
		So(p.RelayName, ShouldEqual, "")

		So(p.RelayIP, ShouldEqual, nil)
		So(p.RelayPort, ShouldEqual, 0)
		So(p.Delay, ShouldEqual, 0.02)
		So(p.Delays.Smtpd, ShouldEqual, 0.01)
		So(p.Delays.Cleanup, ShouldEqual, 0)
		So(p.Delays.Qmgr, ShouldEqual, 0.01)
		So(p.Delays.Smtp, ShouldEqual, 0)
		So(p.Dsn, ShouldEqual, "5.4.4")
		So(p.Status, ShouldEqual, BouncedStatus)
	})
}

func TestQmgrParsing(t *testing.T) {
	Convey("Qmgr expired message", t, func() {
		header, parsed, err := Parse([]byte(`Oct  3 12:39:14 mailhost postfix-12.34.56.78/qmgr[24086]: ` +
			`B54DA300087: from=<redacted@company.com>, status=expired, returned to sender`))

		So(parsed, ShouldNotEqual, nil)
		So(err, ShouldEqual, nil)
		p, cast := parsed.(QmgrReturnedToSender)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 3)
		So(header.Time.Month.String(), ShouldEqual, "October")
		So(header.Time.Hour, ShouldEqual, 12)
		So(header.Time.Minute, ShouldEqual, 39)
		So(header.Time.Second, ShouldEqual, 14)
		So(header.Host, ShouldEqual, "mailhost")
		So(header.Process, ShouldEqual, "qmgr")
		So(p.SenderLocalPart, ShouldEqual, "redacted")
		So(p.SenderDomainPart, ShouldEqual, "company.com")
		So(p.Queue, ShouldEndWith, "B54DA300087")
	})
}
