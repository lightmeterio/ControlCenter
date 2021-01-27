// +build !codeanalysis

package rawparser

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

  ip = ipv4 >setTokBeg %{
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

func parseSmtpdDisconnect(data []byte) (SmtpdDisconnect, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpdDisconnect{}

%%{
	include common "common.rl";

  hostname = [^\[]+ >setTokBeg %{
    r.Host = data[tokBeg:p] 
  }; 

  ip = ipv4 >setTokBeg %{
    r.IP = data[tokBeg:p] 
  };

  main := 'disconnect from ' hostname '[' ip ']'  @{
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

  ip = ipv4 >setTokBeg %{
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