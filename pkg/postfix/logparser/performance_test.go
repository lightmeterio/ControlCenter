// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"bufio"
	"strings"
	"testing"
)

func BenchmarkParser(b *testing.B) {
	content := `Jun  3 10:40:57 mail postfix/submission/smtpd[9708]: connect from some.domain.name[1.2.3.4]
Jun  3 10:40:57 mail postfix/submission/smtpd[9708]: Anonymous TLS connection established from some.domain.name[1.2.3.4]: TLSv1.2 with cipher ECDHE-RSA-AES256-GCM-SHA384 (256/256 bits)
Jun  3 10:40:57 mail postfix/submission/smtpd[9708]: 4AA091855DA0: client=some.domain.name[1.2.3.4], sasl_method=PLAIN, sasl_username=user@sender.com
Jun  3 10:40:57 mail postfix/sender-cleanup/cleanup[9709]: 4AA091855DA0: message-id=<ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com>
Jun  3 10:40:57 mail postfix/sender-cleanup/cleanup[9709]: 4AA091855DA0: replace: header MIME-Version: 1.0 from some.domain.name[1.2.3.4]; from=<user@sender.com> to=<invalid.email@example.com> proto=ESMTP helo=<[192.168.0.178]>: Mime-Version: 1.0
Jun  3 10:40:57 mail opendkim[235]: 4AA091855DA0: DKIM-Signature field added (s=mail, d=mail.sender.com)
Jun  3 10:40:57 mail postfix/qmgr[1005]: 4AA091855DA0: from=<user@sender.com>, size=391, nrcpt=1 (queue active)
Jun  3 10:40:57 mail postfix/submission/smtpd[9708]: disconnect from some.domain.name[1.2.3.4] ehlo=2 starttls=1 auth=1 mail=1 rcpt=1 data=1 quit=1 commands=8
Jun  3 10:40:57 mail postfix/smtpd[9715]: connect from localhost[127.0.0.1]
Jun  3 10:40:57 mail postfix/smtpd[9715]: 776E41855DB2: client=localhost[127.0.0.1]
Jun  3 10:40:57 mail postfix/cleanup[9716]: 776E41855DB2: message-id=<ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com>
Jun  3 10:40:57 mail postfix/qmgr[1005]: 776E41855DB2: from=<user@sender.com>, size=1111, nrcpt=1 (queue active)
Jun  3 10:40:57 mail postfix/smtpd[9715]: disconnect from localhost[127.0.0.1] ehlo=1 mail=1 rcpt=1 data=1 quit=1 commands=5
Jun  3 10:40:57 mail amavis[2279]: (02279-04) Passed CLEAN {RelayedOpenRelay}, [1.2.3.4]:6101 [1.2.3.4] <user@sender.com> -> <invalid.email@example.com>, Queue-ID: 4AA091855DA0, Message-ID: <ca10035e-2951-bfd5-ec7e-1a5773fce1cd@mail.sender.com>, mail_id: 6aLEkMwQa8H2, Hits: -, size: 888, queued_as: 776E41855DB2, 80 ms
Jun  3 10:40:57 mail postfix/smtp[9710]: 4AA091855DA0: to=<invalid.email@example.com>, relay=127.0.0.1[127.0.0.1]:10024, delay=0.23, delays=0.15/0/0/0.08, dsn=2.0.0, status=sent (250 2.0.0 from MTA(smtp:[127.0.0.1]:10025): 250 2.0.0 Ok: queued as 776E41855DB2)
Jun  3 10:40:57 mail postfix/qmgr[1005]: 4AA091855DA0: removed
Jun  3 10:40:57 mail dovecot: imap(user@sender.com)<9755><fJVypiunzhdZ9/xL>: Connection closed (IDLE running for 0.001 + waiting input for 0.001 secs, 2 B in + 10+10 B out, state=wait-input) in=780 out=1920 deleted=0 expunged=0 trashed=0 hdr_count=1 hdr_strings=270 body_count=0 body_strings=0
Jun  3 10:40:58 mail postfix/smtp[9890]: Trusted TLS connection established to mx.example.com[11.22.33.44]:25: TLSv1.2 with cipher ECDHE-RSA-AES256-GCM-SHA384 (256/256 bits)
Jun  3 10:40:59 mail postfix/smtp[9890]: 776E41855DB2: to=<invalid.email@example.com>, relay=mx.example.com[11.22.33.44]:25, delay=1.9, delays=0/0/1.5/0.37, dsn=5.1.1, status=bounced (host mx.example.com[11.22.33.44] said: 550 5.1.1 <invalid.email@example.com> User unknown (in reply to RCPT TO command))
Jun  3 10:40:59 mail postfix/cleanup[9716]: A48191855DA0: message-id=<20200603104059.A48191855DA0@mail.sender.com>
Jun  3 10:40:59 mail postfix/qmgr[1005]: A48191855DA0: from=<>, size=3147, nrcpt=1 (queue active)
Jun  3 10:40:59 mail postfix/bounce[9719]: 776E41855DB2: sender non-delivery notification: A48191855DA0
Jun  3 10:40:59 mail postfix/qmgr[1005]: 776E41855DB2: removed
Jun  3 10:40:59 mail dovecot: lmtp(9956): Connect from local
Jun  3 10:40:59 mail dovecot: lmtp(user@sender.com)<9956><IkiQKTt+117kJgAAWP5Hkg>: msgid=<20200603104059.A48191855DA0@mail.sender.com>: saved mail to INBOX
Jun  3 10:40:59 mail dovecot: lmtp(9956): Disconnect from local: Client has quit the connection (state=READY)
Jun  3 10:40:59 mail postfix/lmtp[9717]: A48191855DA0: to=<user@sender.com>, relay=mail.sender.com[/var/run/dovecot/lmtp], delay=0.03, delays=0.01/0/0.02/0.01, dsn=2.0.0, status=sent (250 2.0.0 <user@sender.com> IkiQKTt+117kJgAAWP5Hkg Saved)
Jun  3 10:40:59 mail postfix/qmgr[1005]: A48191855DA0: removed`

	for i := 0; i < b.N; i++ {
		scanner := bufio.NewScanner(strings.NewReader(content))

		for scanner.Scan() {
			_, _, _ = Parse(scanner.Text())
		}
	}
}
