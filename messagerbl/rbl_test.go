// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package messagerbl

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"net"
	"testing"
	"time"
)

type fakeSettings struct {
	ip net.IP
}

func (s *fakeSettings) IPAddress(context.Context) net.IP {
	return s.ip
}

func TestRBL(t *testing.T) {
	Convey("Test RBL", t, func() {
		ip := net.ParseIP("127.0.0.2")
		settings := &fakeSettings{ip: ip}
		detector := New(settings)
		pub := detector.NewPublisher()

		done, cancel := detector.Run()

		converter := parser.NewTimeConverter(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), func(int, parser.Time, parser.Time) {})

		record := func(s string) postfix.Record {
			h, p, err := parser.Parse([]byte(s))
			So(err, ShouldBeNil)

			return postfix.Record{
				Time:    converter.Convert(h.Time),
				Header:  h,
				Payload: p,
			}
		}

		records := []postfix.Record{
			// Microsoft
			record(`Jan 10 00:00:56 node postfix/smtp[12357]: 375593D395: to=<recipient@example.com>, relay=relay.example.com[254.112.150.90]:25, delay=0.86, delays=0.1/0/0.71/0.05, dsn=5.7.606, status=bounced (host [254.112.150.90] said: 550 5.7.606 Access denied, banned sending IP [239.58.50.50]. To request removal from this list please visit https://sender.office.com/ and follow the directions. For more information please go to  http://go.microsoft.com/fwlink/?LinkID=526655 (AS16012609) (in reply to RCPT TO command))`),
			// No match, status=sent
			record(`Jan 20 11:03:18 server postfix/smtp[30639]: 2847EE008E: to=<info@test2.com>, relay=mail.domain.test[11.22.33.44]:25, delay=6.3, delays=0.04/0.01/6/0.19, dsn=2.1.5, status=sent (250 2.1.5 Ok)`),
			// Mimecast
			record(`Jan 21 14:50:00 mail postfix-135.97.192.135/smtp[25870]: 333E52C4FF72: to=<a.recip@example.com>, relay=us-smtp-inbound-2.mimecast.com[11.22.33.44]:25, delay=1.5, delays=0/0/1.1/0.36, dsn=5.0.0, status=bounced (host us-smtp-inbound-2.mimecast.com[11.22.33.44] said: 550 csi.mimecast.org Poor Reputation Sender. - https://community.mimecast.com/docs/DOC-1369#550 [EM-f6hnWN9maa42OLPquOA.us166] (in reply to RCPT TO command))`),
			// no match, no pattern found
			record(`Jan 22 06:01:00 smtpnode51 postfix-135.97.192.135/smtp[5743]: 4B0D5410DE: to=<h-d66579f3@h-83e81c.com>, relay=h-50f3c4ccac5be32ede8[38.170.83.23]:25, delay=2000, delays=1999/0.13/1.1/0, dsn=4.7.0, status=deferred (host h-50f3c4ccac5be32ede8[38.170.83.23] refused to talk to me: 550 5.7.0 Blocked - see https://h-da21bcc5400cebb4aa45f2/h-fe4ec82fb61b5c4c?ip=135.97.192.135)`),
			// Microsoft
			record(`Feb 18 11:55:37 mail postfix/smtp[19793]: 6BAB0300C9CD: to=<someone@hotmail.com>, relay=hotmail-com.olc.protection.outlook.com[104.47.66.33]:25, delay=0.7, delays=0.04/0.02/0.47/0.16, dsn=5.7.1, status=bounced (host hotmail-com.olc.protection.outlook.com[104.47.66.33] said: 550 5.7.1 Unfortunately, messages from [1.2.3.4] weren't sent. Please contact your Internet service provider since part of their network is on our block list (S3140). You can also refer your provider to http://mail.live.com/mail/troubleshooting.aspx#errors. [MW2NAM12FT005.eop-nam12.prod.protection.outlook.com] (in reply to MAIL FROM command))`),
		}

		for _, r := range records {
			pub.Publish(r)
		}

		resultsChan := make(chan []Result)

		go func() {
			for results := range detector.resultsChan {
				resultsChan <- results.Values[0:results.Size]
			}
		}()

		cancel()
		done()

		results := <-resultsChan

		So(len(results), ShouldEqual, 3)

		So(results[0], ShouldResemble, Result{
			Host:    "Microsoft",
			Address: ip,
			Payload: records[0].Payload.(parser.SmtpSentStatus),
			Header:  records[0].Header,
			Time:    records[0].Time,
		})

		So(results[1], ShouldResemble, Result{
			Host:    "Mimecast",
			Address: net.ParseIP("135.97.192.135"),
			Payload: records[2].Payload.(parser.SmtpSentStatus),
			Header:  records[2].Header,
			Time:    records[2].Time,
		})

		So(results[2], ShouldResemble, Result{
			Host:    "Microsoft",
			Address: ip,
			Payload: records[4].Payload.(parser.SmtpSentStatus),
			Header:  records[4].Header,
			Time:    records[4].Time,
		})
	})
}
