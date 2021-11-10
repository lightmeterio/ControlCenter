// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

import (
  "strings"
)

%% machine dovecotAuthFailedWithReason;
%% write data;

func parseDovecotAuthFailedWithReason(data string) (DovecotAuthFailedWithReason, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	var r DovecotAuthFailedWithReason

%%{
	include common "common.rl";

  dbInfo = [^)]+ >setTokBeg %{
    {
      parts := strings.Split(data[tokBeg:p], ",")
      if len(parts) < 2 {
        return r, false
      }

      r.Username = parts[0]
      r.IP = parts[1]
    }
  };

  # TODO: make this list more generic, so that we can support other mechanisms out of the box
  dbType = ('sql' | 'passwd-file' | 'ldap' | 'pam' | 'lua' | 'bsdauth' | 'shadow' | 'policy') >setTokBeg %{
    r.DB = data[tokBeg:p]
  };

  passwdMismatchMessage = 'Password mismatch' >setTokBeg %{
    r.DovecotAuthFailedReasonPasswordMismatch = data[tokBeg:p]
  };

  unknownUserMessage = 'unknown user' any+ >setTokBeg %{
    r.DovecotAuthFailedReasonUnknownUser = data[tokBeg:p]
  };

  policyServerRefusal = 'Authentication failure due to policy server refusal'  >setTokBeg %{
    r.DovecotAuthFailedReasonAuthPolicyRefusal = data[tokBeg:p]
  };

  extraMessage = any+ >setTokBeg %{
    r.ReasonExplanation = data[tokBeg:p]
  };

  message = (passwdMismatchMessage | unknownUserMessage | (policyServerRefusal (': ' extraMessage)? ));

  main := 'auth: ' dbType '(' dbInfo '): ' message %/{
		return r, true
  };

	write init;
	write exec;
}%%

	return r, false
}
