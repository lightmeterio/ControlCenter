// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

%% machine headerPostfixPart;
%% write data;

func parseHeaderPostfixPart(h *RawHeader, data string) (int, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

%%{
	include common "common.rl";

	hostname = [^ ]+ >setTokBeg %{
		h.Host = data[tokBeg:p]
	};

	# a process name can be postfix or something like /postfix-script
	# but if we read postfix-127.0.0.2, the process name is just 'postfix',
	# as the 127.0.0.2 in this case is the IP address
	processName = '/'? (alnum|'-'|'_')+ >setTokBeg %{
		h.Process = data[tokBeg:p]
	};

	# TODO: support other representations of IP, as well as v6
	processIp = ipv4 >setTokBeg %{
		h.ProcessIP = data[tokBeg:p]
	};

	daemonName = (^']')+ >setTokBeg %{
		h.Daemon = data[tokBeg:p]
	};

	processId = digit+ >setTokBeg %{
		h.ProcessID = data[tokBeg:p]
	};

	processPart = processName ('-' processIp)? ('/' daemonName)?;

	main := hostname ' ' processPart ('[' processId ']')? ': ' @{
		return p, true
	};

	write init;
	write exec;
}%%

	return 0, false
}
