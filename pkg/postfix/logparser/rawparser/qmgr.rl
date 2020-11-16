// +build !codeanalysis

package rawparser

%% machine qmgrReturnedToSender;
%% write data;

func parseQmgrReturnedToSender(data []byte) (QmgrReturnedToSender, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := QmgrReturnedToSender{}

%%{
	include common "common.rl";

	qMgrQueueId = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	senderLocalPart = bracketedEmailLocalPart >setTokBeg %{
		r.SenderLocalPart = normalizeMailLocalPart(data[tokBeg:p])
	};

	senderDomainPart = bracketedEmailDomainPart >setTokBeg %{
		r.SenderDomainPart = data[tokBeg:p]
	};

	main := qMgrQueueId ': from=<' senderLocalPart '@' senderDomainPart '>, status=expired, returned to sender'  @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
