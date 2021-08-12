// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build !codeanalysis

package rawparser

%% machine version;
%% write data;

func parseVersion(data []byte) (Version, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	var r Version

%%{
	include common "common.rl";
	
	postfixDaemonStatus = 'reload' | 'daemon started';
	
	version = (digit|dot)+ >setTokBeg %{
		r = data[tokBeg:p]
	};

	main := postfixDaemonStatus ' -- version ' version ',' any+ @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
