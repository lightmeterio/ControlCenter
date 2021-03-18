// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build !codeanalysis

package rawparser

%% machine cleanupMessageAccepted;
%% write data;

func parseCleanupMessageAccepted(data []byte) (CleanupMessageAccepted, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMessageAccepted{}

%%{
	include common "common.rl";

	queue = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

  bracketedMsgId = [^>]+ >setTokBeg %{
    r.MessageId = data[tokBeg:p]
  };

  freeFormMessageId = [^<>]+ >setTokBeg %{
    r.MessageId = data[tokBeg:p]
  };

  validMsgIdPart = ('<>' | '<' bracketedMsgId '>' | freeFormMessageId);

  corruptedMsgIdPart = ('<' bracketedMsgId | '<' bracketedMsgId '>' [^>]+ | '') %{
    r.Corrupted = true
  };

	main := queue ': message-id=' (validMsgIdPart | corruptedMsgIdPart) %/{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}

%% machine cleanupMilterReject;
%% write data;

func parseCleanupMilterReject(data []byte) (CleanupMilterReject, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMilterReject{}

%%{
	include common "common.rl";

	queue = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	extraMessage = any+ >setTokBeg;

  main := queue ': milter-reject: ' extraMessage @{
		r.ExtraMessage = data[tokBeg:eof]
    return r, true
  };

	write init;
	write exec;
}%%

	return r, false
}
