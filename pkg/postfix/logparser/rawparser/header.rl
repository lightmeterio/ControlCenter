// +build !codeanalysis

package rawparser

%% machine headerPostfixPart;
%% write data;

func parseHeaderPostfixPart(h *RawHeader, data []byte) (int, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

%%{
	action setTokBeg { tokBeg = p }

	hostname = (alnum | '.')+ >setTokBeg %{
		h.Host = data[tokBeg:p]
	};

	processName = alnum+ >setTokBeg %{
		h.Process = data[tokBeg:p]
	};

	processIp = (^'/')+ >setTokBeg %{
		h.ProcessIP = data[tokBeg:p]
	};

	daemonName = (^']')+ >setTokBeg %{
		h.Daemon = data[tokBeg:p]
	};

	processId = digit+ >setTokBeg %{
		h.ProcessID = data[tokBeg:p]
	};

	main := hostname ' ' processName ('-' processIp)? ('/' daemonName)? ('[' processId ']')? ': ' @{
		return p, true
	};

	write init;
	write exec;
}%%

	return 0, false
}
