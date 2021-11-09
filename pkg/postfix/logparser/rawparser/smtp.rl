// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

import (
	"strings"
)

%% machine smtpSentStatusPayload;
%% write data;

func parseSmtpSentStatus(data string) (RawSmtpSentStatus, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := RawSmtpSentStatus{}

%%{
	include common "common.rl";

	smtpQueueId = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	recipientLocalPart = bracketedEmailLocalPart >setTokBeg %{
		r.RecipientLocalPart = normalizeMailLocalPart(data[tokBeg:p])
	};

	recipientDomainPart = bracketedEmailDomainPart >setTokBeg %{
		r.RecipientDomainPart = data[tokBeg:p]
	};

	origRecipientLocalPart = bracketedEmailLocalPart >setTokBeg %{
		r.OrigRecipientLocalPart = normalizeMailLocalPart(data[tokBeg:p])
	};

	origRecipientDomainPart = bracketedEmailDomainPart >setTokBeg %{
		r.OrigRecipientDomainPart = data[tokBeg:p]
	};

	relayName = [^,\[]+ >setTokBeg %{
		r.RelayName = data[tokBeg:p]
	};

	relayIpOrPath = [^\]]+ >setTokBeg %{
		r.RelayIpOrPath = data[tokBeg:p]
	};

	relayPort = digit+ >setTokBeg %{
		r.RelayPort = data[tokBeg:p]
	};

	delay = anythingExceptComma >setTokBeg %{
		r.Delay = data[tokBeg:p]
	};

	delays = anythingExceptComma >setTokBeg %{
		{
			delays := data[tokBeg:p]
			split := strings.Split(delays, string("/"))

			if len(split) != 4 {
				// delays has format 0.0/0.0/0.0/0.0
				return r, false
			}

			r.Delays[0] = delays
			r.Delays[1] = split[0]
			r.Delays[2] = split[1]
			r.Delays[3] = split[2]
			r.Delays[4] = split[3]
		}
	};

	dsn = anythingExceptComma >setTokBeg %{
		r.Dsn = data[tokBeg:p]
	};

	status = ('sent'|'bounced'|'deferred') >setTokBeg %{
		r.Status = data[tokBeg:p]
	};

	extraMessage = any+ >setTokBeg;

	main := smtpQueueId ': to=<' recipientLocalPart '@' recipientDomainPart '>, '
	        ('orig_to=<' origRecipientLocalPart ('@' origRecipientDomainPart)? '>, ')?
	        'relay=' ((relayName ('[' relayIpOrPath ']')? (':' relayPort)?)|'none') ', '
	        'delay=' delay ', delays=' delays ', dsn=' dsn ', status=' status ' ' extraMessage @{
		r.ExtraMessage = data[tokBeg:eof]
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}

%% machine smtpSentStatusExtraMessageSentQueuedPayload;
%% write data;

func parseSmtpSentStatusExtraMessageSentQueued(data string) (SmtpSentStatusExtraMessageSentQueued, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpSentStatusExtraMessageSentQueued{}

%%{
	include common "common.rl";

	smtpCode = digit+ >setTokBeg %{
		r.SmtpCode = data[tokBeg:p]
	};

	dsn = (digit+ dot digit+ dot digit+) > setTokBeg %{
		r.Dsn = data[tokBeg:p]
	};

	ip = ipv4 >setTokBeg %{
		r.IP = data[tokBeg:p]
	};

	port = digit+ >setTokBeg %{
		r.Port = data[tokBeg:p]
	};

	queue = queueId > setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	selfDelivery = (smtpCode ' ' dsn ' from MTA(smtp:[' ip ']:' port '): ');

	main := '(' selfDelivery '250 2.0.0 Ok: queued as ' queue ')' %{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
