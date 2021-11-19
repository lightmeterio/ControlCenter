// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

%% machine pickup;
%% write data;

func parsePickup(data string) (Pickup, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := Pickup{}

%%{
	include common "common.rl";

	pickupQueueId = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	pickupUid = digit+ >setTokBeg %{
		r.Uid = data[tokBeg:p]
	};

	sender = [^>]* >setTokBeg %{
		r.Sender = normalizeMailLocalPart(data[tokBeg:p])
	};

	main := pickupQueueId ': uid=' pickupUid ' from=<' sender '>' @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
