// +build !codeanalysis

package rawparser

import (
  "bytes"
)

%% machine smtpSentStatusPayload;
%% write data;

func parseSmtpSentStatus(data []byte) (RawSmtpSentStatus, bool) {
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

	relayIp = [^,\]]+ >setTokBeg %{
		r.RelayIp = data[tokBeg:p]
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
			split := bytes.Split(delays, []byte("/"))

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
        ('orig_to=<' origRecipientLocalPart '@' origRecipientDomainPart '>, ')?
        'relay=' ((relayName '[' relayIp ']:' relayPort)|'none') ', '
			  'delay=' delay ', delays=' delays ', dsn=' dsn ', status=' status ' ' extraMessage @{
		r.ExtraMessage = data[tokBeg:eof]
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
