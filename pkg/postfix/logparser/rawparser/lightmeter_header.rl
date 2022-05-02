// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

%% machine lightmeter_header;
%% write data;

func parseDumpedHeader(data string) (LightmeterDumpedHeader, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	var r LightmeterDumpedHeader

%%{
	include common "common.rl";

  headerQueueId = queueId >setTokBeg %{
    r.Queue = data[tokBeg:p]
  };

  name = [^"]+ >setTokBeg %{
    r.Key = data[tokBeg:p]
  };

  value = [^"]+ >setTokBeg %{
    r.Value = data[tokBeg:p]
  };
	
	main := headerQueueId ': header name="' name '", value="' value '"' @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
