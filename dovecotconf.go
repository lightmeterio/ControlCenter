// SPDX-FileCopyrightText: 2022 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"fmt"
)

func setupDovecotConfig(dovecotIsOld bool) {
	authTpl := func() string {
		if dovecotIsOld {
			return ""
		}

		return `
# Check Lightmeter blocklist before auth (pre-auth), not after
# Also, report un·successful auth attempts
auth_policy_check_before_auth = yes
auth_policy_check_after_auth = no
auth_policy_report_after_auth = yes
`
	}()

	nonce := func() string {
		// TODO: generate a randoom string every time, although having a constant nonce should not cause
		// us any harm at the moment
		return "ZVKBQYhlZxHWxkJ62hJeTzacEEM7"
	}()

	var tpl = `
# Dovecot will query Lightmeter's blocklist for every incoming IMAP/POP3 connection
auth_policy_server_url = https://auth.intelligence.lightmeter.io/auth

# See https://doc.dovecot.org/settings/core/#core_setting-auth_policy_hash_nonce for more information
auth_policy_hash_nonce = ` + nonce + `

# The remote IP address, that is trying to authenticate, is the minimal bit of information
# needed by Lightmeter to block illegitimate authentication attempts
# See https://doc.dovecot.org/settings/core/#setting-auth-policy-request-attributes for more information
auth_policy_request_attributes = remote=%{rip}

# The following is needed to verify the number of blocked auth attempts
auth_verbose = yes
` + authTpl

	//nolint:forbidigo
	fmt.Println(tpl)
}
