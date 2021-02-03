// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0

package messagerbl

import (
	"regexp"
)

// TODO: make those matches somehow configurable, preferably at runtime
var defaultMatchers = matchers{
	{host: "Microsoft", dsn: "5.7.606", pattern: regexp.MustCompile(`http:\/\/go\.microsoft\.com\/fwlink\/\?LinkID=526655`)},
	{host: "Google", dsn: "4.7.0", pattern: regexp.MustCompile(`421.*https:\/\/support\.google\.com\/mail`)},
	{host: "Google", dsn: "5.7.1", pattern: regexp.MustCompile(`421.*https:\/\/support\.google\.com\/mail`)},
	{host: "AOL", dsn: "5.3.0", pattern: regexp.MustCompile(`DNSBL:ATTRBL 521`)},
	{host: "AOL", pattern: regexp.MustCompile(`delivery temporarily suspended.*refused to talk to me.*mx.aol.com`)},
	{host: "ATT", pattern: regexp.MustCompile(`blocked by sbc:blacklist\.mailrelay\.att\.net\. 521 DNSRBL: Blocked for abuse`)},
	{host: "Mimecast", dsn: "5.0.0", pattern: regexp.MustCompile(`Poor Reputation Sender.*https:\/\/community\.mimecast\.com`)},
	{host: "Mimecast", pattern: regexp.MustCompile(`http:\/\/kb\.mimecast\.com\/Mimecast_Knowledge_Base`)},
	{host: "Yahoo", pattern: regexp.MustCompile(`\[TS03\]`)},
	{host: "Tend Micro", dsn: "5.7.1", pattern: regexp.MustCompile(`blocked using Trend Micro`)},
}
