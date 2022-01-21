// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package messagerbl

import (
	"regexp"
)

// TODO: make those matches somehow configurable, preferably at runtime.
// TODO: Analyze all patterns in parallel, as a single regular expression. This can be efficiently done using Ragel, I believe.
var defaultMatchers = matchers{
	{host: "Microsoft", dsn: "5.7.606", pattern: regexp.MustCompile(`http:\/\/go\.microsoft\.com\/fwlink\/\?LinkID=526655`)},
	{host: "Google", dsn: "4.7.0", pattern: regexp.MustCompile(`421.*https:\/\/support\.google\.com\/mail`)},
	{host: "Google", dsn: "5.7.1", pattern: regexp.MustCompile(`421.*https:\/\/support\.google\.com\/mail`)},
	{host: "Google", dsn: "5.7.1", pattern: regexp.MustCompile(`Our system has detected that.*suspicious.*low reputation.*Please visit.*support\.google\.com`)},
	{host: "Google", dsn: "5.7.1", pattern: regexp.MustCompile(`The user or domain that you are sending.*has a policy.*prohibited.*support\.google\.com`)},
	{host: "Google", dsn: "5.7.1", pattern: regexp.MustCompile(`Our system has detected an.*spam.*blocked.*support\.google\.com\/mail\/\?p=UnsolicitedIPError`)},
	{host: "AOL", dsn: "5.3.0", pattern: regexp.MustCompile(`DNSBL:ATTRBL 521`)},
	{host: "AOL", pattern: regexp.MustCompile(`delivery temporarily suspended.*refused to talk to me.*mx.aol.com`)},
	{host: "ATT", pattern: regexp.MustCompile(`blocked by sbc:blacklist\.mailrelay\.att\.net\. 521 DNSRBL: Blocked for abuse`)},
	{host: "Mimecast", dsn: "5.0.0", pattern: regexp.MustCompile(`Poor Reputation Sender.*https:\/\/community\.mimecast\.com`)},
	{host: "Mimecast", pattern: regexp.MustCompile(`http:\/\/kb\.mimecast\.com\/Mimecast_Knowledge_Base`)},
	{host: "Yahoo", pattern: regexp.MustCompile(`\[TS03\]`)},
	{host: "Trend Micro", dsn: "5.7.1", pattern: regexp.MustCompile(`blocked using Trend Micro`)},
	{host: "Microsoft", dsn: "5.7.1", pattern: regexp.MustCompile(`Unfortunately, messages from .* weren't sent\. Please contact your Internet service provider since part of their network is on our block list \(S3140\)\. You can also refer your provider to http:\/\/mail\.live\.com`)},
	{host: "Microsoft", dsn: "5.7.511", pattern: regexp.MustCompile(`Access denied, banned sender.*. To request removal from this list please forward this message to delist@messaging.microsoft.com\.`)},

	// dsn 4.7.0 (deferred) was reported on gitlab issue #615, and 5.7.0 (bounced) has been found in some of our (obfuscated) logs.
	{host: "iCloud", pattern: regexp.MustCompile(`refused to talk to me.* Blocked - see https:\/\/support\.proofpoint\.com\/dnsbl-lookup\.cgi`)},
}
