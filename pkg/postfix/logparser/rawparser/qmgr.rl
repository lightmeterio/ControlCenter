// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

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

%% machine qmgrMailQueued;
%% write data;

func parseQmgrMailQueued(data []byte) (QmgrMailQueued, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := QmgrMailQueued{}

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

  size = digit+ >setTokBeg %{
    r.Size = data[tokBeg:p]
  };

  nrcpt = digit+ >setTokBeg %{
    r.Nrcpt = data[tokBeg:p]
  };

	main := qMgrQueueId ': from=<' (senderLocalPart '@' senderDomainPart)? '>, size=' size ', nrcpt=' nrcpt ' (queue active)'  @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}


%% machine qmgrRemoved;
%% write data;

func parseQmgrRemoved(data []byte) (QmgrRemoved, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := QmgrRemoved{}

%%{
	include common "common.rl";

	qMgrQueueId = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	main := qMgrQueueId ': removed' @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
