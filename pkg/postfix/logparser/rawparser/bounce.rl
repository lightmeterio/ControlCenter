// +build !codeanalysis

package rawparser

%% machine bounceCreated;
%% write data;

func parseBounceCreated(data []byte) (BounceCreated, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := BounceCreated{}

%%{
	include common "common.rl";

	queue = queueId >setTokBeg %{
		r.Queue = data[tokBeg:p]
	};

	childQueue = queueId >setTokBeg;

	main := queue ': sender ' ('delivery status'|'non-delivery') ' notification: ' childQueue @{
		r.ChildQueue = data[tokBeg:eof]
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
