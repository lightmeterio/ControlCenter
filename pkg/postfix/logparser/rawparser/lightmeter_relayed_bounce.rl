// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

%% machine lightmeter_relayed_bounce;
%% write data;

func parseRelayedBounce(data string) (LightmeterRelayedBounce, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	var r LightmeterRelayedBounce

%%{
	include common "common.rl";

  code = [^"]+ >setTokBeg %{
    r.DeliveryCode = data[tokBeg:p]
  };

  sender = [^>]+ >setTokBeg %{
    r.Sender = data[tokBeg:p]
  };

  recipient = [^>]+ >setTokBeg %{
    r.Recipient = data[tokBeg:p]
  };
	
	mta = [^"]+ >setTokBeg %{
		r.ReportingMTA = data[tokBeg:p]
	};
	
	message = any+ >setTokBeg %{
		r.DeliveryMessage = data[tokBeg:p]
	};
	
	main := 'Bounce: code="' code '", sender=<' sender '>, recipient=<' recipient '>, mta="' mta '", message="' message '"' @{
		return r, true
	};

	write init;
	write exec;
}%%

	return r, false
}
