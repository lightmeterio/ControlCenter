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

  messageIdInBrackets = [^>]* > setTokBeg %{
    r.MessageId = data[tokBeg:p]
  };

  messageIdWithoutBrackets = [^<>' ']+ > setTokBeg @{
    r.MessageId = data[tokBeg:eof]
  };

	main := queue ': message-id=' ('<' messageIdInBrackets '>' | messageIdWithoutBrackets ) @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
