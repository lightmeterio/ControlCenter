// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"errors"
	"net"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/timeutil"
)

func TestErrorKinds(t *testing.T) {
	Convey("Some errors still allows to recover header info", t, func() {
		So(IsRecoverableError(nil), ShouldBeTrue)
		So(IsRecoverableError(ErrUnsupportedLogLine), ShouldBeTrue)
		So(IsRecoverableError(ErrInvalidHeaderLine), ShouldBeFalse)
		So(IsRecoverableError(errors.New("Some Random Error")), ShouldBeFalse)
	})
}

func TestParsingInvalidLines(t *testing.T) {
	Convey("Invalid Line", t, func() {
		_, p, err := Parse(string("Invalid Line"))
		So(p, ShouldBeNil)
		So(err, ShouldEqual, ErrInvalidHeaderLine)
	})
}

func TestParsingUnsupportedGeneralMessage(t *testing.T) {
	Convey("Unsupported Smtp Line", t, func() {
		h, p, err := Parse(string(`Sep 16 00:07:41 smtp-node07.com postfix-10.20.30.40/smtp[31868]: 0D59F4165A:` +
			` host mx-aol.mail.gm0.yahoodns.net[44.55.66.77] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1;i ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command)`))

		So(p, ShouldBeNil)
		So(err, ShouldEqual, ErrUnsupportedLogLine)
		So(h.Process, ShouldEqual, "postfix")
		So(h.Daemon, ShouldEqual, "smtp")
		So(h.Time.Day, ShouldEqual, 16)
		So(h.Time.Month.String(), ShouldEqual, "September")
		So(h.Time.Hour, ShouldEqual, 0)
		So(h.Time.Minute, ShouldEqual, 7)
		So(h.Time.Second, ShouldEqual, 41)
		So(h.Host, ShouldEqual, "smtp-node07.com")
	})

	Convey("Unsupported Log Line", t, func() {
		_, p, err := Parse(string(`Feb 16 00:07:34 smtpnode07 postfix-10.20.30.40/something[2342]: unsupported message`))
		So(p, ShouldBeNil)
		So(err, ShouldEqual, ErrUnsupportedLogLine)
	})

	Convey("Line with leading null chars should remove them and parse properly", t, func() {
		// many leading zero bytes, and then "Jun  6 23:34:18 ucs fetchmail[2174]: Nachricht h-796080212e8fae@h-36a01c19fbb.com@h-615c50f91311c26bed347e:39 von 41 wird gelesen (3157 Bytes im Nachrichtenkopf) (Log-Meldung unvollständig)"
		line := []byte{
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x4a, 0x75, 0x6e, 0x20, 0x20, 0x36,
			0x20, 0x32, 0x33, 0x3a, 0x33, 0x34, 0x3a, 0x31, 0x38, 0x20, 0x75, 0x63,
			0x73, 0x20, 0x66, 0x65, 0x74, 0x63, 0x68, 0x6d, 0x61, 0x69, 0x6c, 0x5b,
			0x32, 0x31, 0x37, 0x34, 0x5d, 0x3a, 0x20, 0x4e, 0x61, 0x63, 0x68, 0x72,
			0x69, 0x63, 0x68, 0x74, 0x20, 0x68, 0x2d, 0x37, 0x39, 0x36, 0x30, 0x38,
			0x30, 0x32, 0x31, 0x32, 0x65, 0x38, 0x66, 0x61, 0x65, 0x40, 0x68, 0x2d,
			0x33, 0x36, 0x61, 0x30, 0x31, 0x63, 0x31, 0x39, 0x66, 0x62, 0x62, 0x2e,
			0x63, 0x6f, 0x6d, 0x40, 0x68, 0x2d, 0x36, 0x31, 0x35, 0x63, 0x35, 0x30,
			0x66, 0x39, 0x31, 0x33, 0x31, 0x31, 0x63, 0x32, 0x36, 0x62, 0x65, 0x64,
			0x33, 0x34, 0x37, 0x65, 0x3a, 0x33, 0x39, 0x20, 0x76, 0x6f, 0x6e, 0x20,
			0x34, 0x31, 0x20, 0x77, 0x69, 0x72, 0x64, 0x20, 0x67, 0x65, 0x6c, 0x65,
			0x73, 0x65, 0x6e, 0x20, 0x28, 0x33, 0x31, 0x35, 0x37, 0x20, 0x42, 0x79,
			0x74, 0x65, 0x73, 0x20, 0x69, 0x6d, 0x20, 0x4e, 0x61, 0x63, 0x68, 0x72,
			0x69, 0x63, 0x68, 0x74, 0x65, 0x6e, 0x6b, 0x6f, 0x70, 0x66, 0x29, 0x20,
			0x28, 0x4c, 0x6f, 0x67, 0x2d, 0x4d, 0x65, 0x6c, 0x64, 0x75, 0x6e, 0x67,
			0x20, 0x75, 0x6e, 0x76, 0x6f, 0x6c, 0x6c, 0x73, 0x74, 0xc3, 0xa4, 0x6e,
			0x64, 0x69, 0x67, 0x29,
		}

		h, p, err := Parse(string(line))
		So(p, ShouldBeNil)
		So(h.Process, ShouldEqual, "fetchmail")
		So(h.Host, ShouldEqual, "ucs")
		So(err, ShouldEqual, ErrUnsupportedLogLine)
	})

	Convey("Unsupported Log Line with slash on process", t, func() {
		h, p, err := Parse(string(`Mar  3 02:55:42 mail process/daemon/with/slash[9708]: any content here`))
		So(err, ShouldEqual, ErrUnsupportedLogLine)
		So(p, ShouldBeNil)
		So(h.Process, ShouldEqual, "process")
		So(h.Daemon, ShouldEqual, "daemon/with/slash")
		So(h.PID, ShouldEqual, 9708)
	})

	Convey("Unsupported Log Line with underscore in the process_name", t, func() {
		h, p, err := Parse(string(`Jun 16 03:39:15 email dk_check[72882]: Starting the dk_check filter...`))
		So(err, ShouldEqual, ErrUnsupportedLogLine)
		So(p, ShouldBeNil)
		So(h.Process, ShouldEqual, "dk_check")
		So(h.Daemon, ShouldEqual, "")
		So(h.PID, ShouldEqual, 72882)
	})

	Convey("Unsupported opendkim line, but time is okay", t, func() {
		h, p, err := Parse(string(`Apr  5 19:00:02 mail opendkim[195]: 407032C4FF6A: DKIM-Signature field added (s=mail, d=lightmeter.io)`))
		So(err, ShouldEqual, ErrUnsupportedLogLine)
		So(p, ShouldBeNil)
		So(h.Process, ShouldEqual, "opendkim")
		So(h.PID, ShouldEqual, 195)
		So(h.Time.Month, ShouldEqual, time.April)
		So(h.Time.Day, ShouldEqual, 5)
		So(h.Time.Hour, ShouldEqual, 19)
		So(h.Time.Minute, ShouldEqual, 0)
		So(h.Time.Second, ShouldEqual, 2)
	})

	Convey("Unsupported dovecot line", t, func() {
		h, p, err := Parse(string(`May  5 18:56:52 mail dovecot: imap(laal@mail.io)<28358><CO3htpid9tRXDNo5>: Connection closed (IDLE running for 0.001 + waiting input for 28.914 secs, 2 B in + 10 B out, state=wait-input) in=703 out=12338 deleted=0 expunged=0 trashed=0 hdr_count=0 hdr_bytes=0 body_count=0 body_bytes=0`))
		So(err, ShouldEqual, ErrUnsupportedLogLine)
		So(p, ShouldBeNil)
		So(h.Process, ShouldEqual, "dovecot")
		So(h.PID, ShouldEqual, 0)
	})

	Convey("Unsupported line starting with slash", t, func() {
		h, p, err := Parse(string(`Dec 17 06:25:48 sm02 /postfix-script[112854]: the Postfix mail system is running: PID: 95072`))
		So(err, ShouldEqual, ErrUnsupportedLogLine)
		So(p, ShouldBeNil)
		So(h.Process, ShouldEqual, "postfix-script")
		So(h.PID, ShouldEqual, 112854)
	})

}

func TestSMTPParsing(t *testing.T) {
	Convey("Basic SMTP Status", t, func() {
		header, parsed, err := Parse(string(`Jun 16 00:07:43 smtpnode07 postfix-10.20.30.40/smtp[3022]: ` +
			`0C31D3D1E6: to=<redacted@aol.com>, relay=mx-aol.mail.gm0.yahoodns.net[11.22.33.44]:25, ` +
			`delay=18910, delays=18900/8.9/0.69/0.03, dsn=4.7.0, status=deferred ` +
			`(host mx-aol.mail.gm0.yahoodns.net[11.22.33.44] said: 421 4.7.0 [TSS04] ` +
			`Messages from 10.20.30.40 temporarily deferred due to user complaints - 4.16.55.1; ` +
			`see https://help.yahoo.com/kb/postmaster/SLN3434.html (in reply to MAIL FROM command))`))
		So(err, ShouldBeNil)
		So(parsed, ShouldNotBeNil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 16)
		So(header.Time.Month, ShouldEqual, time.June)
		So(header.Time.Hour, ShouldEqual, 0)
		So(header.Time.Minute, ShouldEqual, 7)
		So(header.Time.Second, ShouldEqual, 43)
		So(header.Host, ShouldEqual, "smtpnode07")
		So(header.Process, ShouldEqual, "postfix")
		So(header.Daemon, ShouldEqual, "smtp")
		So(header.ProcessIP, ShouldEqual, net.ParseIP("10.20.30.40"))
		So(header.PID, ShouldEqual, 3022)

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
		header, parsed, err := Parse(string(`Jul  5 17:24:35 mail postfix/smtp[9635]: D298F2C60812: to=<"user 1234 with space"@icloud.com>,` +
			` relay=mx6.mail.icloud.com[17.178.97.79]:25, delay=428621, delays=428619/0.02/1.9/0, ` +
			`dsn=4.7.0, status=deferred (host mx6.mail.icloud.com[17.178.97.79] ` +
			`refused to talk to me: 550 5.7.0 Blocked - see https://support.proofpoint.com/dnsbl-lookup.cgi?ip=142.93.169.220)`))
		So(parsed, ShouldNotBeNil)
		So(err, ShouldBeNil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 5)
		So(header.Time.Month, ShouldEqual, time.July)
		So(header.Time.Hour, ShouldEqual, 17)
		So(header.Time.Minute, ShouldEqual, 24)
		So(header.Time.Second, ShouldEqual, 35)
		So(header.Host, ShouldEqual, "mail")
		So(header.Process, ShouldEqual, "postfix")
		So(header.Daemon, ShouldEqual, "smtp")
		So(header.ProcessIP, ShouldBeNil)

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
		header, parsed, err := Parse(string(`Aug  3 04:41:17 mail postfix/smtp[10603]: AE8E32C60819: to=<mail@e.mail.com>, ` +
			`relay=none, delay=0.02, delays=0.01/0/0.01/0, dsn=5.4.4, status=bounced ` +
			`(Host or domain name not found. Name service error for name=e.mail.com type=AAAA: Host not found)`))
		So(parsed, ShouldNotBeNil)
		So(err, ShouldBeNil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 3)
		So(header.Time.Month, ShouldEqual, time.August)
		So(header.Time.Hour, ShouldEqual, 4)
		So(header.Time.Minute, ShouldEqual, 41)
		So(header.Time.Second, ShouldEqual, 17)
		So(header.Host, ShouldEqual, "mail")
		So(header.Process, ShouldEqual, "postfix")
		So(header.Daemon, ShouldEqual, "smtp")

		So(p.Queue, ShouldEqual, "AE8E32C60819")
		So(p.RecipientLocalPart, ShouldEqual, "mail")
		So(p.RecipientDomainPart, ShouldEqual, "e.mail.com")
		So(p.OrigRecipientLocalPart, ShouldEqual, "")
		So(p.OrigRecipientDomainPart, ShouldEqual, "")
		So(p.RelayName, ShouldEqual, "")

		So(p.RelayIP, ShouldBeNil)
		So(p.RelayPort, ShouldEqual, 0)
		So(p.Delay, ShouldEqual, 0.02)
		So(p.Delays.Smtpd, ShouldEqual, 0.01)
		So(p.Delays.Cleanup, ShouldEqual, 0)
		So(p.Delays.Qmgr, ShouldEqual, 0.01)
		So(p.Delays.Smtp, ShouldEqual, 0)
		So(p.Dsn, ShouldEqual, "5.4.4")
		So(p.Status, ShouldEqual, BouncedStatus)
	})

	Convey("Log line with extra message: queued", t, func() {
		Convey("Optional orig_to filled", func() {
			header, parsed, err := Parse(string(`May  5 00:00:00 mail postfix/smtp[17709]: AB5501855DA0: to=<to@mail.com>, ` +
				`orig_to=<orig_to@example.com>, relay=127.0.0.1[127.0.0.1]:10024, delay=0.87, delays=0.68/0.01/0/0.18, ` +
				`dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 2F01D1855DB2)`))
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)
			p, cast := parsed.(SmtpSentStatus)
			So(cast, ShouldBeTrue)

			So(header.Time.Day, ShouldEqual, 5)
			So(header.Time.Month, ShouldEqual, time.May)
			So(header.Time.Hour, ShouldEqual, 0)
			So(header.Time.Minute, ShouldEqual, 0)
			So(header.Time.Second, ShouldEqual, 0)
			So(header.Host, ShouldEqual, "mail")
			So(header.Process, ShouldEqual, "postfix")
			So(header.Daemon, ShouldEqual, "smtp")

			So(p.Queue, ShouldEqual, "AB5501855DA0")
			So(p.RecipientLocalPart, ShouldEqual, "to")
			So(p.RecipientDomainPart, ShouldEqual, "mail.com")
			So(p.OrigRecipientLocalPart, ShouldEqual, "orig_to")
			So(p.OrigRecipientDomainPart, ShouldEqual, "example.com")
			So(p.RelayName, ShouldEqual, "127.0.0.1")
			So(p.Status, ShouldEqual, SentStatus)
			So(p.ExtraMessage, ShouldEqual, `(250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 2F01D1855DB2)`)

			e, cast := p.ExtraMessagePayload.(SmtpSentStatusExtraMessageSentQueued)
			So(cast, ShouldBeTrue)
			So(e.Port, ShouldEqual, 10025)
			So(e.IP, ShouldEqual, net.ParseIP("127.0.0.1"))
			So(e.Queue, ShouldEqual, "2F01D1855DB2")

			// obtained from the beginning of the line, as the values from the end are hard-coded by postfix
			So(e.SmtpCode, ShouldEqual, 250)
			So(e.Dsn, ShouldEqual, "2.0.0")
		})

		Convey("A short extra message that looks like a local delivery", func() {
			_, parsed, err := Parse(string(`Jan 25 20:11:27 mx postfix/smtp[8038]: D1CB62E0A23: to=<h-5c3@h-092c585d.com>, ` +
				`relay=h-bf6f84bb0157e81a7fa40b[135.55.127.35]:25, delay=0.61, delays=0.21/0.01/0.28/0.11, dsn=2.0.0, status=sent ` +
				`(250 2.0.0 Ok: queued as 5744140325)`))
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)
			p, cast := parsed.(SmtpSentStatus)
			So(cast, ShouldBeTrue)

			So(p.Queue, ShouldEqual, "D1CB62E0A23")
			So(p.RecipientLocalPart, ShouldEqual, "h-5c3")
			So(p.RecipientDomainPart, ShouldEqual, "h-092c585d.com")
			So(p.RelayName, ShouldEqual, "h-bf6f84bb0157e81a7fa40b")
			So(p.RelayPort, ShouldEqual, 25)
			So(p.Status, ShouldEqual, SentStatus)
			So(p.ExtraMessage, ShouldEqual, `(250 2.0.0 Ok: queued as 5744140325)`)

			_, cast = p.ExtraMessagePayload.(SmtpSentStatusExtraMessageSentQueued)
			So(cast, ShouldBeFalse)
		})
	})

	Convey("Log line with optional orig_to as root", t, func() {
		_, parsed, err := Parse(string(`May  5 00:00:00 mail postfix/smtp[17709]: AB5501855DA0: to=<to@mail.com>, ` +
			`orig_to=<root>, relay=127.0.0.1[127.0.0.1]:10024, delay=0.87, delays=0.68/0.01/0/0.18, ` +
			`dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 2F01D1855DB2)`))
		So(err, ShouldBeNil)
		So(parsed, ShouldNotBeNil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldBeTrue)

		So(p.OrigRecipientLocalPart, ShouldEqual, "root")
		So(p.OrigRecipientDomainPart, ShouldEqual, "")
	})
}

func TestQmgrParsing(t *testing.T) {
	Convey("Qmgr expired message", t, func() {
		header, parsed, err := Parse(string(`Sep  3 12:39:14 mailhost postfix-12.34.56.78/qmgr[24086]: ` +
			`B54DA300087: from=<redacted@company.com>, status=force-expired, returned to sender`))

		So(parsed, ShouldNotBeNil)
		So(err, ShouldBeNil)
		p, cast := parsed.(QmgrMessageExpired)
		So(cast, ShouldEqual, true)

		So(header.Time.Day, ShouldEqual, 3)
		So(header.Time.Month, ShouldEqual, time.September)
		So(header.Time.Hour, ShouldEqual, 12)
		So(header.Time.Minute, ShouldEqual, 39)
		So(header.Time.Second, ShouldEqual, 14)
		So(header.Host, ShouldEqual, "mailhost")
		So(header.Process, ShouldEqual, "postfix")
		So(header.Daemon, ShouldEqual, "qmgr")
		So(p.SenderLocalPart, ShouldEqual, "redacted")
		So(p.SenderDomainPart, ShouldEqual, "company.com")
		So(p.Queue, ShouldEndWith, "B54DA300087")
		So(p.Message, ShouldEndWith, "returned to sender")
	})
}

func TestPipe(t *testing.T) {
	Convey("Pipe has the same struct as smtp delivery message", t, func() {
		Convey("Example1", func() {
			_, parsed, err := Parse(string(`Jan 25 19:25:19 mx postfix/pipe[6221]: 03EAC2E0EDD: to=<h-7abde52c2@h-ffd2115d4f.com>, relay=dovecot, delay=5.5, delays=5.3/0.01/0/0.16, dsn=2.0.0, status=sent (delivered via dovecot service)`))
			So(parsed, ShouldNotBeNil)
			So(err, ShouldBeNil)
			p, cast := parsed.(SmtpSentStatus)
			So(cast, ShouldBeTrue)

			So(p.Queue, ShouldEqual, "03EAC2E0EDD")
			So(p.RecipientLocalPart, ShouldEqual, "h-7abde52c2")
			So(p.RecipientDomainPart, ShouldEqual, "h-ffd2115d4f.com")
			So(p.RelayName, ShouldEqual, "dovecot")
			So(p.Status, ShouldEqual, SentStatus)
			So(p.ExtraMessage, ShouldEqual, `(delivered via dovecot service)`)
		})
	})
}

func TestLmtpParsing(t *testing.T) {
	Convey("Lmtp uses the same struct as SmtpSentStatus", t, func() {
		Convey("Example1", func() {
			header, parsed, err := Parse(string(`Jan 10 16:15:30 mail postfix/lmtp[11996]: 400643011B47: to=<recipient@example.com>, ` +
				`relay=relay.example.com[/var/run/dovecot/lmtp], delay=0.06, delays=0.02/0.02/0.01/0.01, dsn=2.0.0, status=sent ` +
				`(250 2.0.0 <recipient@example.com> hz3kESIo+1/dLgAAWP5Hkg Saved)`))
			So(parsed, ShouldNotBeNil)
			So(err, ShouldBeNil)
			p, cast := parsed.(SmtpSentStatus)
			So(cast, ShouldBeTrue)

			So(header.Time.Day, ShouldEqual, 10)
			So(header.Time.Month, ShouldEqual, time.January)
			So(header.Time.Hour, ShouldEqual, 16)
			So(header.Time.Minute, ShouldEqual, 15)
			So(header.Time.Second, ShouldEqual, 30)
			So(header.Host, ShouldEqual, "mail")
			So(header.Process, ShouldEqual, "postfix")
			So(header.Daemon, ShouldEqual, "lmtp")

			So(p.Queue, ShouldEqual, "400643011B47")
			So(p.RecipientLocalPart, ShouldEqual, "recipient")
			So(p.RecipientDomainPart, ShouldEqual, "example.com")
			So(p.RelayName, ShouldEqual, "relay.example.com")
			So(p.RelayPath, ShouldEqual, "/var/run/dovecot/lmtp")
			So(p.Status, ShouldEqual, SentStatus)
			So(p.ExtraMessage, ShouldEqual, `(250 2.0.0 <recipient@example.com> hz3kESIo+1/dLgAAWP5Hkg Saved)`)
		})

		Convey("Example2", func() {
			_, parsed, err := Parse(string(`Feb  1 13:17:08 mail postfix/lmtp[28699]: AED441541AFC: to=<h-06819@h-b4e62aa55116.com>, relay=h-ac7182297368ddc0f[private/dovecot-lmtp], delay=0.21, delays=0.07/0.01/0.01/0.12, dsn=2.0.0, status=sent (250 2.0.0 <h-06819@h-b4e62aa55116.com> wArTL0TxF2AccAAAYr7Zvw Saved)`))
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)
			p, cast := parsed.(SmtpSentStatus)
			So(cast, ShouldBeTrue)
			So(p.Queue, ShouldEqual, "AED441541AFC")
		})

	})
}

func TestTimeConversion(t *testing.T) {
	Convey("Convert to Unix timestamp on UTC", t, func() {
		t := Time{Day: 25, Month: time.May, Hour: 5, Minute: 12, Second: 22}
		ts := t.Unix(2008, time.UTC)
		So(ts, ShouldEqual, 1211692342)
	})
}

func TestCleanupProcessing(t *testing.T) {
	Convey("Cleanup", t, func() {
		Convey("e-mail like message-id", func() {
			_, payload, err := Parse(string("Jun  3 10:40:57 mail postfix/sender-cleanup/cleanup[9709]: 4AA091855DA0: message-id=<ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com>"))
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.MessageId, ShouldEqual, "ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com")
			So(p.Corrupted, ShouldBeFalse)
		})

		Convey("free form like message", func() {
			_, payload, err := Parse(string("Jun  3 10:40:57 mail postfix/cleanup[9709]: 4AA091855DA0: message-id=656587JHGJHG"))
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.Corrupted, ShouldBeFalse)
			So(p.MessageId, ShouldEqual, "656587JHGJHG")
		})

		Convey("free form like message, with a datetime?!", func() {
			_, payload, err := Parse(string(`Jan 25 10:00:04 mx postfix/cleanup[21978]: AF98F2E0768: message-id=2021-01-25 10:00:05.006274`))
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.Corrupted, ShouldBeFalse)
			So(p.MessageId, ShouldEqual, `2021-01-25 10:00:05.006274`)
		})

		Convey("A very suspicious line, with text after the bracket", func() {
			_, payload, err := Parse(string(`Jan 26 04:27:34 mx postfix/cleanup[1525]: 59DC92E2BB8: message-id=<h-22b35ef986ae4024dffa0@h-70550910e99892.com>+29A6080B6290BF43`))
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.Corrupted, ShouldBeTrue)
			So(p.MessageId, ShouldEqual, `h-22b35ef986ae4024dffa0@h-70550910e99892.com`)
		})

		Convey("totally empty messageid. it should not happen, but does. and it's an error", func() {
			_, payload, err := Parse(string(`Jan 29 17:27:21 mx postfix/cleanup[22582]: A55EA2E01B5: message-id=`))
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.Corrupted, ShouldBeTrue)
			So(p.MessageId, ShouldEqual, ``)
		})

		Convey("corrupted bracketed messageid", func() {
			// maybe this was an attack attempt (the messageid can be set via smtp by the sender),
			// or just syslog that cropped the message. Who knows?
			_, payload, err := Parse(string(`Jan 26 15:50:21 mx postfix/cleanup[22403]: 1A3642E0C5C: message-id=<eyJyZXBseV90byI6Im5ldWZlcnRAaW5ub3YuZW5lcmd5IiwiYnhfbXNnX2lkIjoiODUwNmViOTYtMzNlNS00ZWI4LWE4NzItZDI2NjBiMWMyZWJiIiwiYnhfY29tcGFueV9pZCI6ImpwdHdkdjNxNWVzayIsInN1YmplY3QiOiJBbmdlYm90IHNhbGlkb21vIDkgb2huZSBOb3RzdHJvbS9VU1YgfCBQcm9qZWt0IERvYmxlciIsImJvZHkiOiJTZWhyIGdlZWhydGUvciBIZXJyIFJlbsOpIELDvGNoaSxcclxuXHJcbnZpZWxlbiBEYW5rIGbDvHIgSWhyZSBBbmZyYWdlIHVuZCBkYXMgZGFtaXQgdmVyYnVuZGVuZSBJbnRlcmVzc2UgYW4gdW5zZXJlbSBzYWxpZG9tby1TYWx6YmF0dGVyaWVzcGVpY2hlcnN5c3RlbS4gXHJcblxyXG5HZXJuZSB1bnRlcmJyZWl0ZW4gd2lyIElobmVuIHVuc2VyIEFuZ2Vib3QgS0QtQUdCLTIwMjEtMDEtMjYtMTAzMDkwLTAyMSB2b20gMjYuMDEuMjAyMSDDvGJlciBDSEYgMTcnNjQxLjI2LlxyXG5cclxuVW50ZXIgZm9sZ2VuZGVtIExpbmsga8O2bm5lbiBTaWUgZGFzIGdlc2FtdGUgQW5nZWJvdCBhbnNlaGVuOlxyXG5odHRwczovL25ldHdvcmsuYmV4aW8uY29tL29mZmVyLzU0MGQyYTcyNzBiYTE2YjFjNGQwZTgzMWVmOTA4YjZjZmU4YWFmOGFhZWQwZDBkMzliYzdhOGZkNzY5ZjY1Y2RcclxuXHJcblp1ciBBdXNsw7ZzdW5nIGRlcyBBdWZ0cmFncyBiZXN0w6R0aWdlbiBTaWUgYml0dGUgaW0gSGVhZGVyIGRlcyBPbmxpbmUtQW5nZWJvdGVzIGRlbiBBa3plcHRpZXJlb? i1CdXR0b24uIFNpZSBlcmhhbHRlbiBkYW5uIHVtZ2VoZW5kIHZvbiB1bnMgZWluZSBBdWZ0cmFnc2Jlc3TDpHRpZ3VuZy5cclxuXHJcbldpciBob2ZmZW4sIGRhc3MgZGFzIEFuZ2Vib3QgSWhyZW4gQW5mb3JkZXJ1bmdlbiBlbnRzcHJpY2h0IHVuZCBmcmV1ZW4gdW5zIGF1ZiBkaWUgWnVzYW1tZW5hcmJlaXQuIEbDvHIgUsO8Y2tmcmFnZW4gdW5kIHdlaXRlcmUgSW5mb3JtYXRpb25lbiBzdGVoZW4gd2lyIGdlcm4genVyIFZlcmbDvGd1bmc6IFxyXG5vZmZlcnRlQGlubm92LmVuZXJneSBvZGVyICs0MSAzMyA1NTIgMTAgMTBcclxuXHJcbk1pdCBmcmV1bmRsaWNoZW4gR3LDvHNzZW5cclxuXHJcbk1heCBVcnNpbiAvIFBldGVyIFJ1dGhcclxuSW5ub3ZlbmVyZ3kgR21iSFxyXG5fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX1xyXG5cclxuV2VpdGVyZSBJbmZvcm1hdGlvbmVuIHp1IGlubm92ZW5lcmd5IHVuZCBkZW4gU2FsemJhdHRlcmllc3BlaWNoZXJsw7ZzdW5nZW4gZmluZGVuIFNpZSB1bnRlcjogXHJcbnd3dy5pbm5vdi5lbmVyZ3lcclxuX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX18iLCJzbXRwX2lkIjoiPDE2NjE5NzExMDMuNzEyLjE2MTE2NzI2MTkzMTdAZW1haWwtc2VydmljZS05Nm? Y3ZmI`))
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.MessageId, ShouldEqual, `eyJyZXBseV90byI6Im5ldWZlcnRAaW5ub3YuZW5lcmd5IiwiYnhfbXNnX2lkIjoiODUwNmViOTYtMzNlNS00ZWI4LWE4NzItZDI2NjBiMWMyZWJiIiwiYnhfY29tcGFueV9pZCI6ImpwdHdkdjNxNWVzayIsInN1YmplY3QiOiJBbmdlYm90IHNhbGlkb21vIDkgb2huZSBOb3RzdHJvbS9VU1YgfCBQcm9qZWt0IERvYmxlciIsImJvZHkiOiJTZWhyIGdlZWhydGUvciBIZXJyIFJlbsOpIELDvGNoaSxcclxuXHJcbnZpZWxlbiBEYW5rIGbDvHIgSWhyZSBBbmZyYWdlIHVuZCBkYXMgZGFtaXQgdmVyYnVuZGVuZSBJbnRlcmVzc2UgYW4gdW5zZXJlbSBzYWxpZG9tby1TYWx6YmF0dGVyaWVzcGVpY2hlcnN5c3RlbS4gXHJcblxyXG5HZXJuZSB1bnRlcmJyZWl0ZW4gd2lyIElobmVuIHVuc2VyIEFuZ2Vib3QgS0QtQUdCLTIwMjEtMDEtMjYtMTAzMDkwLTAyMSB2b20gMjYuMDEuMjAyMSDDvGJlciBDSEYgMTcnNjQxLjI2LlxyXG5cclxuVW50ZXIgZm9sZ2VuZGVtIExpbmsga8O2bm5lbiBTaWUgZGFzIGdlc2FtdGUgQW5nZWJvdCBhbnNlaGVuOlxyXG5odHRwczovL25ldHdvcmsuYmV4aW8uY29tL29mZmVyLzU0MGQyYTcyNzBiYTE2YjFjNGQwZTgzMWVmOTA4YjZjZmU4YWFmOGFhZWQwZDBkMzliYzdhOGZkNzY5ZjY1Y2RcclxuXHJcblp1ciBBdXNsw7ZzdW5nIGRlcyBBdWZ0cmFncyBiZXN0w6R0aWdlbiBTaWUgYml0dGUgaW0gSGVhZGVyIGRlcyBPbmxpbmUtQW5nZWJvdGVzIGRlbiBBa3plcHRpZXJlb? i1CdXR0b24uIFNpZSBlcmhhbHRlbiBkYW5uIHVtZ2VoZW5kIHZvbiB1bnMgZWluZSBBdWZ0cmFnc2Jlc3TDpHRpZ3VuZy5cclxuXHJcbldpciBob2ZmZW4sIGRhc3MgZGFzIEFuZ2Vib3QgSWhyZW4gQW5mb3JkZXJ1bmdlbiBlbnRzcHJpY2h0IHVuZCBmcmV1ZW4gdW5zIGF1ZiBkaWUgWnVzYW1tZW5hcmJlaXQuIEbDvHIgUsO8Y2tmcmFnZW4gdW5kIHdlaXRlcmUgSW5mb3JtYXRpb25lbiBzdGVoZW4gd2lyIGdlcm4genVyIFZlcmbDvGd1bmc6IFxyXG5vZmZlcnRlQGlubm92LmVuZXJneSBvZGVyICs0MSAzMyA1NTIgMTAgMTBcclxuXHJcbk1pdCBmcmV1bmRsaWNoZW4gR3LDvHNzZW5cclxuXHJcbk1heCBVcnNpbiAvIFBldGVyIFJ1dGhcclxuSW5ub3ZlbmVyZ3kgR21iSFxyXG5fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX1xyXG5cclxuV2VpdGVyZSBJbmZvcm1hdGlvbmVuIHp1IGlubm92ZW5lcmd5IHVuZCBkZW4gU2FsemJhdHRlcmllc3BlaWNoZXJsw7ZzdW5nZW4gZmluZGVuIFNpZSB1bnRlcjogXHJcbnd3dy5pbm5vdi5lbmVyZ3lcclxuX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX19fX18iLCJzbXRwX2lkIjoiPDE2NjE5NzExMDMuNzEyLjE2MTE2NzI2MTkzMTdAZW1haWwtc2VydmljZS05Nm? Y3ZmI`)
			So(p.Corrupted, ShouldBeTrue)
		})

		Convey("Submission cleanup (gitlab issue #558)", func() {
			_, payload, err := Parse(`Sep 28 11:21:18 mail2 postfix/submission/cleanup[30571]: DD3B7144: message-id=<h-e3dc9fb6dfc97a822113dc127f54fcc2f322@h-00eed68f.com>`)
			So(err, ShouldBeNil)
			p, cast := payload.(CleanupMessageAccepted)
			So(cast, ShouldBeTrue)
			So(p.Corrupted, ShouldBeFalse)
			So(p.MessageId, ShouldEqual, `h-e3dc9fb6dfc97a822113dc127f54fcc2f322@h-00eed68f.com`)
		})
	})
}

func TestPickup(t *testing.T) {
	Convey("Pickup", t, func() {
		_, payload, err := Parse(string(`Feb  1 13:17:02 mail postfix/pickup[28541]: 08ACF1541B01: uid=42 from=<someone>`))
		So(err, ShouldBeNil)
		p, cast := payload.(Pickup)
		So(cast, ShouldBeTrue)
		So(p.Queue, ShouldEqual, `08ACF1541B01`)
		So(p.Uid, ShouldEqual, 42)
		So(p.Sender, ShouldEqual, "someone")
	})
}

func TestLocalDaemon(t *testing.T) {
	Convey("Pickup", t, func() {
		_, payload, err := Parse(string(`Jun 20 05:02:07 ns4 postfix/local[16460]: 95154657C: to=<h-493fac8f3@h-ea3f4afa.com>, orig_to=<h-195704c@h-20b651e8120a33ec11.com>, relay=local, delay=0.1, delays=0.09/0/0/0.01, dsn=2.0.0, status=sent (delivered to command: procmail -a "$EXTENSION" DEFAULT=$HOME/Maildir/)`))
		So(err, ShouldBeNil)
		p, cast := payload.(SmtpSentStatus)
		So(cast, ShouldBeTrue)
		So(p.RecipientLocalPart, ShouldEqual, "h-493fac8f3")
		So(p.OrigRecipientLocalPart, ShouldEqual, "h-195704c")
		So(p.RecipientDomainPart, ShouldEqual, "h-ea3f4afa.com")
		So(p.OrigRecipientDomainPart, ShouldEqual, "h-20b651e8120a33ec11.com")
		So(p.Queue, ShouldEqual, "95154657C")
	})
}

func TestSmtpdConnect(t *testing.T) {
	Convey("Test smtpd Connect", t, func() {
		Convey("ipv4", func() {
			_, payload, err := Parse(string(`Jan 25 20:17:06 mx postfix/smtpd[3946]: connect from unknown[224.93.112.97]`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdConnect)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`224.93.112.97`))
		})

		Convey("ipv6", func() {
			_, payload, err := Parse(string(`Jan 25 20:12:52 mx postfix/smtps/smtpd[8377]: connect from unknown[1002:1712:4e2b:d061:5dff:19f:c85f:a48f]`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdConnect)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`1002:1712:4e2b:d061:5dff:19f:c85f:a48f`))
		})
	})
}

func TestSmtpdDisconnect(t *testing.T) {
	Convey("Test smtpd Disconnect", t, func() {
		Convey("ipv4", func() {
			_, payload, err := Parse(string(`Jan 25 20:17:06 mx postfix/smtpd[3946]: disconnect from unknown[224.93.112.97] ehlo=1 auth=0/1 rset=1 quit=1 commands=3/4`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdDisconnect)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`224.93.112.97`))
		})

		Convey("ipv6", func() {
			_, payload, err := Parse(string(`Jan 25 20:12:52 mx postfix/smtps/smtpd[8377]: disconnect from unknown[1002:1712:4e2b:d061:5dff:19f:c85f:a48f] ehlo=1 auth=0/1 rset=1 quit=1 commands=3/4`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdDisconnect)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`1002:1712:4e2b:d061:5dff:19f:c85f:a48f`))
		})

		Convey("Several params", func() {
			_, payload, err := Parse(string(`Jul 13 17:41:40 mail postfix/smtpd[26098]: disconnect from unknown[11.22.33.44] ehlo=1 auth=8/14 mail=1 rcpt=0/1 data=0/1 rset=1 commands=3/19`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdDisconnect)
			So(cast, ShouldBeTrue)
			So(p.Stats, ShouldResemble, map[string]SmtpdDisconnectStat{
				"ehlo":     {Success: 1, Total: 1},
				"auth":     {Success: 8, Total: 14},
				"mail":     {Success: 1, Total: 1},
				"rcpt":     {Success: 0, Total: 1},
				"data":     {Success: 0, Total: 1},
				"rset":     {Success: 1, Total: 1},
				"commands": {Success: 3, Total: 19},
			})
		})

	})
}

func TestSmtpdMailAccepted(t *testing.T) {
	Convey("Test smtpd mail accepted", t, func() {
		Convey("ipv4", func() {
			_, payload, err := Parse(string(`Jan 25 06:32:43 mx postfix/smtpd[26382]: 477832E0134: client=h-a4984b7e1cf68ab295d77c5df3[224.93.112.97]`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdMailAccepted)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`224.93.112.97`))
		})

		Convey("ipv6", func() {
			_, payload, err := Parse(string(`Jan 25 06:32:43 mx postfix/smtpd[26382]: 477832E0134: client=h-a4984b7e1cf68ab295d77c5df3[2aff:d7f:d:a::aaa]`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdMailAccepted)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`2aff:d7f:d:a::aaa`))
		})

		Convey("ipv6, short", func() {
			_, payload, err := Parse(string(`Jan 25 18:50:11 mx postfix/smtpd[5178]: 0DAFB2E2D27: client=localhost[::1]`))
			So(err, ShouldBeNil)
			p, cast := payload.(SmtpdMailAccepted)
			So(cast, ShouldBeTrue)
			So(p.IP, ShouldEqual, net.ParseIP(`::1`))
		})
	})
}

func TestLongQueueId(t *testing.T) {
	Convey("Long queue id (gitlab issue #504)", t, func() {
		_, payload, err := Parse(string(`Jan 25 06:32:43 mx postfix/smtpd[26382]: 3Pt2mN2VXxznjll: client=h-a4984b7e1cf68ab295d77c5df3[224.93.112.97]`))
		So(err, ShouldBeNil)
		p, cast := payload.(SmtpdMailAccepted)
		So(cast, ShouldBeTrue)
		So(p.Queue, ShouldEqual, "3Pt2mN2VXxznjll")
	})
}

func TestCleanupMilterReject(t *testing.T) {
	Convey("Milter reject", t, func() {
		_, payload, err := Parse(string(`Jan 25 18:54:51 mx postfix/cleanup[8966]: B37FD2E05B9: milter-reject: END-OF-MESSAGE from h-ca74a0a011076cd81347f8f11e[254.65.43.194]: 4.7.1 Try again later; from=<bounce+1b6a63.922c68-user=h-ffd2115d4f@h-79b594737831a5d176dabf9.com> to=<h-7abde52c2@h-ffd2115d4f.com> proto=ESMTP helo=<h-ca74a0a011076cd81347f8f11e>`))
		So(err, ShouldBeNil)
		p, cast := payload.(CleanupMilterReject)
		So(cast, ShouldBeTrue)
		So(p.Queue, ShouldEqual, "B37FD2E05B9")
		So(p.ExtraMessage, ShouldEqual, `END-OF-MESSAGE from h-ca74a0a011076cd81347f8f11e[254.65.43.194]: 4.7.1 Try again later; from=<bounce+1b6a63.922c68-user=h-ffd2115d4f@h-79b594737831a5d176dabf9.com> to=<h-7abde52c2@h-ffd2115d4f.com> proto=ESMTP helo=<h-ca74a0a011076cd81347f8f11e>`)
	})
}

func TestSmtpdReject(t *testing.T) {
	Convey("Smtpd reject", t, func() {
		_, payload, err := Parse(string(`Feb  8 21:28:47 mx postfix/smtps/smtpd[1036]: DE81A2E2DAA: reject: RCPT from unknown[2a02:168:636a::15e2]: 550 5.1.1 <h-c715634009216@h-14dc4a6d.com>: Recipient address rejected: User unknown in virtual mailbox table; from=<h-d2315d@h-24e89d.com> to=<h-c715634009216@h-14dc4a6d.com> proto=ESMTP helo=<[IPv6:2a02:168:636a::15e2]>`))
		So(err, ShouldBeNil)
		p, cast := payload.(SmtpdReject)
		So(cast, ShouldBeTrue)
		So(p.Queue, ShouldEqual, "DE81A2E2DAA")
		So(p.ExtraMessage, ShouldEqual, `RCPT from unknown[2a02:168:636a::15e2]: 550 5.1.1 <h-c715634009216@h-14dc4a6d.com>: Recipient address rejected: User unknown in virtual mailbox table; from=<h-d2315d@h-24e89d.com> to=<h-c715634009216@h-14dc4a6d.com> proto=ESMTP helo=<[IPv6:2a02:168:636a::15e2]>`)
	})
}

func TestRFC3339Time(t *testing.T) {
	Convey("RFC3339 time", t, func() {
		h, payload, err := ParseWithCustomTimeFormat(string(`2021-05-16T00:01:42.278515+02:00 hq5 postfix/qmgr[21496]: 0262E27A61D7: from=<h-1b6694c3@h-6f3118263bf.com>, size=19314, nrcpt=1 (queue active)`), timeutil.RFC3339TimeFormat{})
		So(err, ShouldBeNil)
		p, cast := payload.(QmgrMailQueued)
		So(cast, ShouldBeTrue)
		So(p.Queue, ShouldEqual, "0262E27A61D7")

		So(h.Time.Year, ShouldEqual, 2021)
		So(h.Time.Month, ShouldEqual, 5)
		So(h.Time.Day, ShouldEqual, 16)
		So(h.Time.Hour, ShouldEqual, 0)
		So(h.Time.Minute, ShouldEqual, 1)
		So(h.Time.Second, ShouldEqual, 42)
	})
}

func TestVirtualParsing(t *testing.T) {
	Convey("Virtual delivery uses the same syntax as smtp delivery", t, func() {
		header, parsed, err := Parse(string(`Jul 25 06:17:23 mail postfix/virtual[2438]: 9BD26E0D25: to=<reci@pient.com>, orig_to=<orig@recipient.com>, relay=virtual, delay=1.3, delays=1.2/0.02/0/0.04, dsn=2.0.0, status=sent (delivered to maildir)`))
		So(parsed, ShouldNotBeNil)
		So(err, ShouldBeNil)
		p, cast := parsed.(SmtpSentStatus)
		So(cast, ShouldBeTrue)

		So(header.Time.Day, ShouldEqual, 25)
		So(header.Time.Month, ShouldEqual, time.July)
		So(header.Time.Hour, ShouldEqual, 6)
		So(header.Time.Minute, ShouldEqual, 17)
		So(header.Time.Second, ShouldEqual, 23)
		So(header.Host, ShouldEqual, "mail")
		So(header.Process, ShouldEqual, "postfix")
		So(header.Daemon, ShouldEqual, "virtual")

		So(p.Queue, ShouldEqual, "9BD26E0D25")
		So(p.RecipientLocalPart, ShouldEqual, "reci")
		So(p.RecipientDomainPart, ShouldEqual, "pient.com")
		So(p.OrigRecipientLocalPart, ShouldEqual, "orig")
		So(p.OrigRecipientDomainPart, ShouldEqual, "recipient.com")
		So(p.RelayName, ShouldEqual, "virtual")
		So(p.Status, ShouldEqual, SentStatus)
		So(p.ExtraMessage, ShouldEqual, `(delivered to maildir)`)
	})
}

func TestDovecotLogParsing(t *testing.T) {
	Convey("Dovecot authentication failure", t, func() {
		Convey("sql passwd mismatch", func() {
			header, parsed, err := Parse(`Jun 11 13:57:17 main dovecot: auth: sql(admin@example.ru,192.168.144.226,<6rXunFtu493AqJDi>): Password mismatch`)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)

			So(header.Time.Day, ShouldEqual, 11)
			So(header.Time.Month, ShouldEqual, time.June)
			So(header.Host, ShouldEqual, "main")
			So(header.Process, ShouldEqual, "dovecot")
			So(header.Daemon, ShouldEqual, "")

			p, cast := parsed.(DovecotAuthFailed)
			So(cast, ShouldBeTrue)

			So(p.DB, ShouldEqual, "sql")
			So(p.Username, ShouldEqual, "admin@example.ru")
			So(p.IP, ShouldResemble, net.ParseIP("192.168.144.226"))
			So(p.Reason, ShouldEqual, DovecotAuthFailedReasonPasswordMismatch)
		})

		Convey("passwd file unknown user", func() {
			_, parsed, err := Parse(`Oct 11 09:30:51 mail dovecot: auth: passwd-file(alice,1.2.3.4): unknown user (SHA1 of given password: 011c94)`)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)

			p, cast := parsed.(DovecotAuthFailed)
			So(cast, ShouldBeTrue)

			So(p.DB, ShouldEqual, "passwd-file")
			So(p.Username, ShouldEqual, "alice")
			So(p.IP, ShouldResemble, net.ParseIP("1.2.3.4"))
			So(p.Reason, ShouldEqual, DovecotAuthFailedReasonUnknownUser)
		})

		Convey("connection blocked, no extra message", func() {
			_, parsed, err := Parse(`Nov  6 18:07:37 mail dovecot: auth: policy(alice,33.44.55.66): Authentication failure due to policy server refusal`)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)

			p, cast := parsed.(DovecotAuthFailed)
			So(cast, ShouldBeTrue)

			So(p.DB, ShouldEqual, "policy")
			So(p.Username, ShouldEqual, "alice")
			So(p.IP, ShouldResemble, net.ParseIP("33.44.55.66"))
			So(p.Reason, ShouldEqual, DovecotAuthFailedReasonAuthPolicyRefusal)
			So(p.ReasonExplanation, ShouldEqual, "")
		})

		Convey("connection blocked, with extra message", func() {
			_, parsed, err := Parse(`Nov  6 18:07:37 mail dovecot: auth: policy(alice,33.44.55.66): Authentication failure due to policy server refusal: Blocked for real`)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)

			p, cast := parsed.(DovecotAuthFailed)
			So(cast, ShouldBeTrue)

			So(p.DB, ShouldEqual, "policy")
			So(p.Username, ShouldEqual, "alice")
			So(p.IP, ShouldResemble, net.ParseIP("33.44.55.66"))
			So(p.Reason, ShouldEqual, DovecotAuthFailedReasonAuthPolicyRefusal)
			So(p.ReasonExplanation, ShouldEqual, "Blocked for real")
		})

		Convey("Parse In-Reply-To header created by our milter", func() {
			_, parsed, err := Parse(`Jan 20 19:48:04 teupos lightmeter/headers[161]: B9996EABB6: header name="In-Reply-To", value="<da454dd13590a0a65a3f492eb2c3932134c4f81cc7f452f1ee2452e0aa06411b@example.com>"`)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)

			p, cast := parsed.(LightmeterDumpedHeader)
			So(cast, ShouldBeTrue)

			So(p.Queue, ShouldEqual, `B9996EABB6`)
			So(p.Key, ShouldEqual, "In-Reply-To")
			So(p.Value, ShouldEqual, "<da454dd13590a0a65a3f492eb2c3932134c4f81cc7f452f1ee2452e0aa06411b@example.com>")
		})

		Convey("Parse relayed-bounce log line created by our milter", func() {
			_, parsed, err := Parse(`Apr 25 05:19:38 lightmetermail lightmeter/relayed-bounce[54]: Bounce: code="5.6.7", sender=<sender@domain.org>, recipient=<recipient@other.com>, mta="some.mta.net", message="550 5.6.7 DNS domain ''lala.com'' does not exist [Message=InfoDomainNonexistent] [LastAttemptedServerName=lala.com] [blah]"`)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)

			p, cast := parsed.(LightmeterRelayedBounce)
			So(cast, ShouldBeTrue)

			So(p.Sender, ShouldEqual, `sender@domain.org`)
			So(p.Recipient, ShouldEqual, "recipient@other.com")
			So(p.DeliveryCode, ShouldEqual, "5.6.7")
			So(p.DeliveryMessage, ShouldEqual, `550 5.6.7 DNS domain ''lala.com'' does not exist [Message=InfoDomainNonexistent] [LastAttemptedServerName=lala.com] [blah]`)
			So(p.ReportingMTA, ShouldEqual, "some.mta.net")
		})
	})
}
