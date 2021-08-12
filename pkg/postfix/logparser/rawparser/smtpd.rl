// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build !codeanalysis

package rawparser

import "strconv"

func mustConvertToInt(s []byte) int {
	v, err := strconv.Atoi(string(s))
  if err != nil {
    // NOTE: only use it when you know the input is really a number!
    panic(err)
  }
  return v
}


%% machine smtpdConnect;
%% write data;

func parseSmtpdConnect(data []byte) (SmtpdConnect, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpdConnect{}

%%{
	include common "common.rl";

  hostname = [^\[]+ >setTokBeg %{
    r.Host = data[tokBeg:p]
  };

  ip = squareBracketedValue >setTokBeg %{
    r.IP = data[tokBeg:p]
  };

  main := 'connect from ' hostname '[' ip ']' @{
    return r, true
  };

	write init;
	write exec;
}%%

	return r, false
}

%% machine smtpdDisconnect;
%% write data;

func smtpdDisconnectStatFromValues(composed bool, firstValue, secondValue int) SmtpdDisconnectStat {
  if composed {
    return SmtpdDisconnectStat{Success: firstValue, Total: secondValue}
  }

  return SmtpdDisconnectStat{Success: firstValue, Total: firstValue}
}

func parseSmtpdDisconnect(data []byte) (SmtpdDisconnect, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpdDisconnect{Stats: map[string]SmtpdDisconnectStat{}}

  var (
    currentStatKey string
    firstStat int
    secondStat int
    composedStat bool
  )

%%{
	include common "common.rl";

  hostname = [^\[]+ >setTokBeg %{
    r.Host = data[tokBeg:p]
  };

  ip = squareBracketedValue >setTokBeg %{
    r.IP = data[tokBeg:p]
  };

  first_stat = digit+ >setTokBeg %{
    firstStat = mustConvertToInt(data[tokBeg:p])
  };

  second_stat = digit+ >setTokBeg %{
    composedStat = true
    secondStat = mustConvertToInt(data[tokBeg:p])
  };

  # NOTE: it's this way to avoid ambiguity as both kinds of value (X and X/Y have the same prefix)
  stat_value = first_stat ("/" second_stat)?;

  stat_key = alpha+ >setTokBeg %{
    currentStatKey = string(data[tokBeg:p])
  };

  stat = ' ' stat_key '=' stat_value %{
    r.Stats[currentStatKey] = smtpdDisconnectStatFromValues(composedStat, firstStat, secondStat)
    firstStat = 0
    secondStat = 0
    composedStat = false
  };

  main := 'disconnect from ' hostname '[' ip ']' stat+ %/{
    return r, true
  };

  write init;
  write exec;

}%%

  return r, false
}

%% machine smtpdMailAccepted;
%% write data;

// TODO: accept additional metadata (sasl_method and sasl_username)
func parseSmtpdMailAccepted(data []byte) (SmtpdMailAccepted, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpdMailAccepted{}

%%{
	include common "common.rl";

  queue = queueId >setTokBeg %{
    r.Queue = data[tokBeg:p]
  };

  hostname = [^\[]+ >setTokBeg %{
    r.Host = data[tokBeg:p]
  };

  ip = squareBracketedValue >setTokBeg %{
    r.IP = data[tokBeg:p]
  };

  main := queue ': client=' hostname '[' ip ']' @{
    return r, true
  };

  write init;
  write exec;

}%%

  return r, false
}

%% machine smtpdReject;
%% write data;

// TODO: accept additional metadata (sasl_method and sasl_username)
func parseSmtpdReject(data []byte) (SmtpdReject, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpdReject{}

%%{
	include common "common.rl";

  queue = queueId >setTokBeg %{
    r.Queue = data[tokBeg:p]
  };

  extraMessage = any+ >setTokBeg;

  main := queue ': reject: ' extraMessage @{
    r.ExtraMessage = data[tokBeg:eof]
    return r, true
  };

  write init;
  write exec;
}%%

  return r, false
}
