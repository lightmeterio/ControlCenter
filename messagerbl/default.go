package messagerbl

import (
	"gitlab.com/lightmeter/controlcenter/settings/globalsettings"
)

// TODO: make those matches somehow configurable, preferably at runtime
var defaultMatchers = matchers{
	{host: "Microsoft", dsn: "5.7.606", pattern: pattern(`http:\/\/go\.microsoft\.com\/fwlink\/\?LinkID=526655`)},
	{host: "Google", dsn: "4.7.0", pattern: pattern(`421.*https:\/\/support\.google\.com\/mail`)},
	{host: "Google", dsn: "5.7.1", pattern: pattern(`421.*https:\/\/support\.google\.com\/mail`)},
	{host: "AOL", dsn: "5.3.0", pattern: pattern(`DNSBL:ATTRBL 521`)},
	{host: "AOL", pattern: pattern(`delivery temporarily suspended.*refused to talk to me.*mx.aol.com`)},
	{host: "ATT", pattern: pattern(`blocked by sbc:blacklist\.mailrelay\.att\.net\. 521 DNSRBL: Blocked for abuse`)},
	{host: "Mimecast", dsn: "5.0.0", pattern: pattern(`Poor Reputation Sender.*https:\/\/community\.mimecast\.com`)},
	{host: "Mimecast", pattern: pattern(`http:\/\/kb\.mimecast\.com\/Mimecast_Knowledge_Base`)},
	{host: "Yahoo", pattern: pattern(`\[TS03\]`)},
	{host: "Tend Micro", dsn: "5.7.1", pattern: pattern(`blocked using Trend Micro`)},
}

func New(settings globalsettings.Getter) *Detector {
	return newDetectorWithMatchers(settings, defaultMatchers)
}
