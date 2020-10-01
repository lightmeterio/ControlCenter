package localrbl

// List obtained from http://multirbl.valli.org/list/"
var DefaultRBLs = []string{
	"0spamtrust.fusionzero.com",             // 0spam DNSWL
	"bl.0spam.org",                          // 0spam General DNSBL Listings
	"0spam.fusionzero.com",                  // 0spam General DNSBL Listings (mirror)
	"nbl.0spam.org",                         // 0spam Network DNSBL Listings
	"0spam-n.fusionzero.com",                // 0spam Network DNSBL Listings (mirror)
	"url.0spam.org",                         // 0spam URLBL Listings
	"0spamurl.fusionzero.com",               // 0spam URLBL Listings (mirror)
	"uribl.zeustracker.abuse.ch",            // abuse.ch ZeuS Tracker Domain
	"ipbl.zeustracker.abuse.ch",             // abuse.ch ZeuS Tracker IP
	"contacts.abuse.net",                    // Abuse.net
	"rbl.abuse.ro",                          // abuse.ro IP RBL
	"uribl.abuse.ro",                        // abuse.ro URI RBL
	"abuse-contacts.abusix.org",             // abusix.org Abuse Contact DB
	"spam.dnsbl.anonmails.de",               // anonmails.de DNSBL
	"dnsbl.anticaptcha.net",                 // AntiCaptcha.NET IPv4
	"dnsbl6.anticaptcha.net",                // AntiCaptcha.NET IPv6
	"orvedb.aupads.org",                     // ANTISPAM-UFRJ orvedb
	"rsbl.aupads.org",                       // ANTISPAM-UFRJ rsbl
	"block.ascams.com",                      // Ascams.com Block
	"superblock.ascams.com",                 // Ascams.com Superblock
	"aspews.ext.sorbs.net",                  // ASPEWS Listings
	"ips.backscatterer.org",                 // Backscatterer.org
	"b.barracudacentral.org",                // Barracuda Reputation Block List
	"bb.barracudacentral.org",               // Barracuda Reputation Block List (for SpamAssassin)
	"list.bbfh.org",                         // BBFH Level 1
	"l1.bbfh.ext.sorbs.net",                 // BBFH Level 1 (@SORBS)
	"l2.bbfh.ext.sorbs.net",                 // BBFH Level 2 (@SORBS)
	"l3.bbfh.ext.sorbs.net",                 // BBFH Level 3 (@SORBS)
	"l4.bbfh.ext.sorbs.net",                 // BBFH Level 4 (@SORBS)
	"all.ascc.dnsbl.bit.nl",                 // BIT.nl all ascc IPv4 address space list
	"all.v6.ascc.dnsbl.bit.nl",              // BIT.nl all ascc IPv6 address space list
	"all.dnsbl.bit.nl",                      // BIT.nl all IPv4 address space list
	"ipv6.all.dnsbl.bit.nl",                 // BIT.nl all IPv6 address space list
	"bitonly.dnsbl.bit.nl",                  // BIT.nl own IPv4 and IPv6 address space list
	"blackholes.tepucom.nl",                 // blackholes.tepucom.nl
	"blacklist.netcore.co.in",               // blacklist.netcore.co.in
	"rbl.blakjak.net",                       // BlakJak.net RBL
	"netscan.rbl.blockedservers.com",        // BlockedServers NetScan RBL
	"rbl.blockedservers.com",                // BlockedServers RBL
	"spam.rbl.blockedservers.com",           // BlockedServers Spam RBL
	"list.blogspambl.com",                   // Blog Spam Blacklist
	"bsb.empty.us",                          // Blog Spam Blocklist (empty.us)
	"bsb.spamlookup.net",                    // Blog Spam Blocklist (spamlookup.net)
	"query.bondedsender.org",                // Bondedsender
	"plus.bondedsender.org",                 // Bondedsender plus
	"dnsbl1.dnsbl.borderware.com",           // borderware.com DNSBL1
	"dnsbl2.dnsbl.borderware.com",           // borderware.com DNSBL2
	"dnsbl3.dnsbl.borderware.com",           // borderware.com DNSBL3
	"dul.dnsbl.borderware.com",              // borderware.com DUL
	"black.dnsbl.brukalai.lt",               // Brukalai.lt DNSBL black
	"light.dnsbl.brukalai.lt",               // Brukalai.lt DNSBL light
	"white.dnsbl.brukalai.lt",               // Brukalai.lt DNSBL white
	"blacklist.sci.kun.nl",                  // C&CZ's own black list
	"whitelist.sci.kun.nl",                  // C&CZ's own white list
	"dul.blackhole.cantv.net",               // cantv.net dul
	"hog.blackhole.cantv.net",               // cantv.net hog
	"rhsbl.blackhole.cantv.net",             // cantv.net rhsbl
	"rot.blackhole.cantv.net",               // cantv.net rot
	"spam.blackhole.cantv.net",              // cantv.net spam
	"cbl.abuseat.org",                       // CBL
	"rbl.choon.net",                         // choon.net IPv4 DNSBL
	"rwl.choon.net",                         // choon.net IPv4 DNSWL
	"ipv6.rbl.choon.net",                    // choon.net IPv6 DNSBL
	"ipv6.rwl.choon.net",                    // choon.net IPv6 DNSWL
	"zz.countries.nerd.dk",                  // countries.nerd.dk DNSBL (zz)
	"dnsbl.cyberlogic.net",                  // Cyberlogic DNSBL
	"bogons.cymru.com",                      // Cymru Bogon List
	"v4.fullbogons.cymru.com",               // Cymru Fullbogon IPv4 List
	"v6.fullbogons.cymru.com",               // Cymru Fullbogon IPv6 List
	"origin.asn.cymru.com",                  // Cymru origin IPv4 asn list
	"origin6.asn.cymru.com",                 // Cymru origin IPv6 asn list
	"peer.asn.cymru.com",                    // Cymru peer asn list
	"tor.dan.me.uk",                         // dan.me.uk (all tor nodes)
	"torexit.dan.me.uk",                     // dan.me.uk (only tor exit nodes)
	"dnsbl.darklist.de",                     // darklist.de
	"openproxy.bls.digibase.ca",             // Digibase BLS Open Proxy
	"proxyabuse.bls.digibase.ca",            // Digibase BLS Proxy Abuse
	"spambot.bls.digibase.ca",               // Digibase BLS Spambot
	"rbl.dns-servicios.com",                 // DNS-SERVICIOS RBL
	"dnsbl.abyan.es",                        // dnsbl.abyan.es
	"dnsbl.beetjevreemd.nl",                 // dnsbl.beetjevreemd.nl
	"dnsbl.calivent.com.pe",                 // dnsbl.calivent.com.pe
	"dnsbl.isx.fr",                          // dnsbl.isx.fr
	"dnsbl.mcu.edu.tw",                      // dnsbl.mcu.edu.tw
	"dnsbl.net.ua",                          // dnsbl.net.ua
	"dnsbl.rv-soft.info",                    // dnsbl.rv-soft.info
	"dnsblchile.org",                        // dnsblchile.org
	"dwl.dnswl.org",                         // DNSWL.org Domain Whitelist
	"list.dnswl.org",                        // SWL.org IP Whitelist
	"vote.drbl.caravan.ru",                  // DRBL caravan.ru (vote node)
	"work.drbl.caravan.ru",                  // DRBL caravan.ru (work node)
	"vote.drbl.gremlin.ru",                  // DRBL gremlin.ru (vote node)
	"work.drbl.gremlin.ru",                  // DRBL gremlin.ru (work node)
	"bl.drmx.org",                           // DrMX
	"dnsbl.dronebl.org",                     // DroneBL
	"rbl.efnet.org",                         // EFnet RBL
	"rbl.efnetrbl.org",                      // EFnet RBL mirror
	"tor.efnet.org",                         // EFnet TOR
	"rbl.fasthosts.co.uk",                   // Fasthosts RBL
	"bl.fmb.la",                             // fmb.la bl
	"communicado.fmb.la",                    // fmb.la communicado
	"nsbl.fmb.la",                           // fmb.la nsbl
	"sa.fmb.la",                             // fmb.la sa
	"short.fmb.la",                          // fmb.la short
	"fnrbl.fast.net",                        // fnrbl.fast.net
	"forbidden.icm.edu.pl",                  // forbidden.icm.edu.pl
	"88.blocklist.zap",                      // Frontbridge’s 88.blocklist.zap
	"hil.habeas.com",                        // Habeas Infringer List
	"accredit.habeas.com",                   // Habeas SafeList
	"sa-accredit.habeas.com",                // Habeas SafeList (for SpamAssassin)
	"hul.habeas.com",                        // Habeas User List
	"sohul.habeas.com",                      // Habeas User List (including Non-Verified-Optin)
	"hostkarma.junkemailfilter.com",         // stkarma
	"black.junkemailfilter.com",             // Hostkarma blacklist
	"nobl.junkemailfilter.com",              // Hostkarma no blacklist
	"dnsbl.cobion.com",                      // IBM DNS Blacklist
	"spamrbl.imp.ch",                        // ImproWare IP based spamlist
	"wormrbl.imp.ch",                        // ImproWare IP based wormlist
	"dnsbl.inps.de",                         // inps.de-DNSBL
	"dnswl.inps.de",                         // inps.de-DNSWL
	"rbl.interserver.net",                   // InterServer BL
	"rbl.iprange.net",                       // IPrange.net RBL
	"iadb.isipp.com",                        // ISIPP Accreditation Database
	"iadb2.isipp.com",                       // ISIPP Accreditation Database (IADB2)
	"iddb.isipp.com",                        // ISIPP Accreditation Database (IDDB)
	"wadb.isipp.com",                        // ISIPP Accreditation Database (WADB)
	"whitelist.rbl.ispa.at",                 // ISPA (Internet Service Provider Austria) Whitelist
	"mail-abuse.blacklist.jippg.org",        // JIPPG's RBL Project (mail-abuse Listings)
	"dnsbl.justspam.org",                    // JustSpam.org
	"dnsbl.kempt.net",                       // Kempt.net DNS Black List
	"spamlist.or.kr",                        // KISA-RBL
	"bl.konstant.no",                        // KONSTANT DNSBL
	"krn.korumail.com",                      // KoruMail Reputation Network (KRN)
	"admin.bl.kundenserver.de",              // kundenserver.de admin.bl
	"relays.bl.kundenserver.de",             // kundenserver.de relays
	"schizo-bl.kundenserver.de",             // kundenserver.de schizo-bl
	"spamblock.kundenserver.de",             // kundenserver.de spamblock
	"worms-bl.kundenserver.de",              // kundenserver.de worms-bl
	"spamguard.leadmon.net",                 // Leadmon.Net's SpamGuard Listings (LNSG)
	"rbl.lugh.ch",                           // lugh.ch DNSBL
	"dnsbl.madavi.de",                       // Madavi:BL
	"niprbl.mailcleaner.net",                // MailCleaner NIPRBL
	"uribl.mailcleaner.net",                 // MailCleaner URIBL
	"blacklist.mailrelay.att.net",           // mailrelay.att.net blacklist
	"bl.mailspike.net",                      // Mailspike Blacklist
	"rep.mailspike.net",                     // Mailspike Reputation
	"wl.mailspike.net",                      // Mailspike Whitelist
	"z.mailspike.net",                       // Mailspike Zero-hour Data
	"bl.mav.com.br",                         // MAV BL
	"cidr.bl.mcafee.com",                    // McAfee RBL
	"dnsbl.forefront.microsoft.com",         // Microsoft Forefront DNSBL
	"bl.mipspace.com",                       // MIPSpace
	"combined.rbl.msrbl.net",                // MSRBL combined
	"images.rbl.msrbl.net",                  // MSRBL images
	"phishing.rbl.msrbl.net",                // MSRBL phishing
	"spam.rbl.msrbl.net",                    // MSRBL spam
	"virus.rbl.msrbl.net",                   // MSRBL virus
	"web.rbl.msrbl.net",                     // MSRBL web
	"relays.nether.net",                     // nether.net (relays)
	"trusted.nether.net",                    // nether.net (trusted)
	"unsure.nether.net",                     // nether.net (unsure)
	"ix.dnsbl.manitu.net",                   // NiX Spam DNSBL
	"dbl.nordspam.com",                      // NordSpam Domain Blacklist
	"bl.nordspam.com",                       // NordSpam IP Blacklist
	"bl.nosolicitado.org",                   // NoSolicitado.org BL
	"bl.worst.nosolicitado.org",             // NoSolicitado.org Worst BL
	"wl.nszones.com",                        // nsZones.com DNSWL
	"dyn.nszones.com",                       // nsZones.com Dyn
	"sbl.nszones.com",                       // nsZones.com SBL
	"bl.nszones.com",                        // nsZones.com SBL+Dyn
	"ubl.nszones.com",                       // nsZones.com SURBL
	"bl.octopusdns.com",                     // Octopus RBL Monster
	"blacklist.mail.ops.asp.att.net",        // ops.asp.att.net blacklist mail
	"blacklist.sequoia.ops.asp.att.net",     // ops.asp.att.net blacklist sequoia
	"spam.pedantic.org",                     // Pedantic.org spam
	"pofon.foobar.hu",                       // pofon.foobar.hu IP Blacklist
	"ispmx.pofon.foobar.hu",                 // pofon.foobar.hu ISP mail relay whitelist
	"uribl.pofon.foobar.hu",                 // pofon.foobar.hu URI Blacklist
	"bl.rbl.polspam.pl",                     // Polspam BL
	"bl-h1.rbl.polspam.pl",                  // Polspam BL-H1
	"bl-h2.rbl.polspam.pl",                  // Polspam BL-H2
	"bl-h3.rbl.polspam.pl",                  // Polspam BL-H3
	"bl-h4.rbl.polspam.pl",                  // Polspam BL-H4
	"bl6.rbl.polspam.pl",                    // Polspam BL6
	"cnkr.rbl.polspam.pl",                   // Polspam CNKR
	"dyn.rbl.polspam.pl",                    // Polspam Dyn
	"ip4.white.polspam.pl",                  // Polspam IPv4 Whitelist
	"ip6.white.polspam.pl",                  // Polspam IPv6 Whitelist
	"lblip4.rbl.polspam.pl",                 // Polspam LBLIP4
	"lblip6.rbl.polspam.pl",                 // Polspam LBLIP6
	"rblip4.rbl.polspam.pl",                 // Polspam RBLIP4
	"rblip6.rbl.polspam.pl",                 // Polspam RBLIP6
	"rhsbl.rbl.polspam.pl",                  // Polspam RHSBL
	"rhsbl-h.rbl.polspam.pl",                // Polspam RHSBL-H
	"safe.dnsbl.prs.proofpoint.com",         // Proofpoint Dynamic Reputation
	"psbl.surriel.com",                      // PSBL (Passive Spam Block List)
	"whitelist.surriel.com",                 // PSBL whitelist
	"rbl.rbldns.ru",                         // rbl.rbldns.ru
	"rbl.schulte.org",                       // rbl.schulte.org
	"rbl.zenon.net",                         // rbl.zenon.net
	"rbl.realtimeblacklist.com",             // realtimeBLACKLIST.COM
	"access.redhawk.org",                    // Redhawk.org
	"eswlrev.dnsbl.rediris.es",              // RedIRIS ListaBlanca ESWL
	"mtawlrev.dnsbl.rediris.es",             // RedIRIS ListaBlanca MTAWL
	"abuse.rfc-clueless.org",                // RFC-Clueless (RFC²) abuse RBL
	"bogusmx.rfc-clueless.org",              // RFC-Clueless (RFC²) BogusMX RBL
	"dsn.rfc-clueless.org",                  // RFC-Clueless (RFC²) DSN RBL
	"elitist.rfc-clueless.org",              // RFC-Clueless (RFC²) Elitist RBL
	"fulldom.rfc-clueless.org",              // RFC-Clueless (RFC²) Metalist RBL
	"postmaster.rfc-clueless.org",           // RFC-Clueless (RFC²) postmaster RBL
	"whois.rfc-clueless.org",                // RFC-Clueless (RFC²) whois RBL
	"mailsl.dnsbl.rjek.com",                 // rjek.com mailsl DNSBL
	"urlsl.dnsbl.rjek.com",                  // rjek.com urlsl DNSBL
	"asn.routeviews.org",                    // Route Views Project asn
	"aspath.routeviews.org",                 // Route Views Project aspath
	"dnsbl.rymsho.ru",                       // Rymsho's DNSBL
	"rhsbl.rymsho.ru",                       // Rymsho's RHSBL
	"all.s5h.net",                           // s5h.net RBL
	"public.sarbl.org",                      // SARBL
	"rhsbl.scientificspam.net",              // scientificspam.net Domain list
	"bl.scientificspam.net",                 // scientificspam.net IP list
	"reputation-domain.rbl.scrolloutf1.com", // Scrollout F1 Reputation Domain
	"reputation-ip.rbl.scrolloutf1.com",     // Scrollout F1 Reputation IP
	"reputation-ns.rbl.scrolloutf1.com",     // Scrollout F1 Reputation NS
	"query.senderbase.org",                  // SenderBase®
	"sa.senderbase.org",                     // SenderBase® (for SpamAssassin)
	"rf.senderbase.org",                     // SenderBase® (Reputation List)
	"bl.score.senderscore.com",              // SenderScore Blacklist
	"score.senderscore.com",                 // SenderScore Reputationlist
	"singular.ttk.pte.hu",                   // SINGULARis Spam/scam blocklist
	"blackholes.scconsult.com",              // Solid Clues Blacklist
	"dnsbl.sorbs.net",                       // SORBS Aggregate zone
	"problems.dnsbl.sorbs.net",              // SORBS Aggregate zone (problems)
	"proxies.dnsbl.sorbs.net",               // SORBS Aggregate zone (proxies)
	"relays.dnsbl.sorbs.net",                // SORBS Aggregate zone (relays)
	"safe.dnsbl.sorbs.net",                  // SORBS Aggregate zone (safe)
	"nomail.rhsbl.sorbs.net",                // SORBS Domain names indicating no email sender
	"badconf.rhsbl.sorbs.net",               // SORBS Domain names pointing to bad addresses
	"dul.dnsbl.sorbs.net",                   // SORBS Dynamic IP Addresses
	"zombie.dnsbl.sorbs.net",                // SORBS hijacked networks
	"block.dnsbl.sorbs.net",                 // SORBS Hosts demanding never be tested by SORBS
	"escalations.dnsbl.sorbs.net",           // SORBS netblocks of spam supporting service providers
	"http.dnsbl.sorbs.net",                  // SORBS Open HTTP Proxies
	"misc.dnsbl.sorbs.net",                  // SORBS Open other Proxies
	"smtp.dnsbl.sorbs.net",                  // SORBS Open SMTP relays
	"socks.dnsbl.sorbs.net",                 // SORBS Open SOCKS Proxies
	"rhsbl.sorbs.net",                       // SORBS RHS Aggregate zone
	"spam.dnsbl.sorbs.net",                  // SORBS Spamhost (any time)
	"recent.spam.dnsbl.sorbs.net",           // SORBS Spamhost (last 28 days)
	"new.spam.dnsbl.sorbs.net",              // SORBS Spamhost (last 48 hours)
	"old.spam.dnsbl.sorbs.net",              // SORBS Spamhost (last year)
	"web.dnsbl.sorbs.net",                   // SORBS Vulnerable formmailers
	"korea.services.net",                    // South Korean Network Blocking List
	"geobl.spameatingmonkey.net",            // Spam Eating Monkey GeoBL (deny all)
	"origin.asn.spameatingmonkey.net",       // Spam Eating Monkey SEM-ASN-ORIGIN
	"backscatter.spameatingmonkey.net",      // Spam Eating Monkey SEM-BACKSCATTER
	"bl.spameatingmonkey.net",               // Spam Eating Monkey SEM-BLACK
	"fresh.spameatingmonkey.net",            // Spam Eating Monkey SEM-FRESH
	"fresh10.spameatingmonkey.net",          // Spam Eating Monkey SEM-FRESH10
	"fresh15.spameatingmonkey.net",          // Spam Eating Monkey SEM-FRESH15
	"fresh30.spameatingmonkey.net",          // Spam Eating Monkey SEM-FRESH30
	"freshzero.spameatingmonkey.net",        // Spam Eating Monkey SEM-FRESHZERO
	"bl.ipv6.spameatingmonkey.net",          // Spam Eating Monkey SEM-IPV6BL
	"netbl.spameatingmonkey.net",            // Spam Eating Monkey SEM-NETBLACK
	"uribl.spameatingmonkey.net",            // Spam Eating Monkey SEM-URI
	"urired.spameatingmonkey.net",           // Spam Eating Monkey SEM-URIRED
	"netblockbl.spamgrouper.to",             // Spam Grouper Net block list
	"all.spam-rbl.fr",                       // Spam-RBL.fr
	"bl.spamcop.net",                        // SpamCop Blocking List
	"sbl.spamdown.org",                      // Spamdown RBL
	"dbl.spamhaus.org",                      // Spamhaus DBL Domain Block List
	"_vouch.dwl.spamhaus.org",               // Spamhaus DWL Domain Whitelist
	"pbl.spamhaus.org",                      // amhaus PBL Policy Block List
	"sbl.spamhaus.org",                      // amhaus SBL Spamhaus Block List
	"sbl-xbl.spamhaus.org",                  // Spamhaus SBL-XBL Combined Block List
	"swl.spamhaus.org",                      // Spamhaus SWL IP Whitelist
	"xbl.spamhaus.org",                      // amhaus XBL Exploits Block List
	"zen.spamhaus.org",                      // amhaus ZEN Combined Block List
	"feb.spamlab.com",                       // SpamLab FEB
	"rbl.spamlab.com",                       // SpamLab RBL
	"all.spamrats.com",                      // SpamRATS! all
	"auth.spamrats.com",                     // SpamRATS! Auth
	"dyna.spamrats.com",                     // SpamRATS! Dyna
	"noptr.spamrats.com",                    // SpamRATS! NoPtr
	"spam.spamrats.com",                     // SpamRATS! Spam
	"spamsources.fabel.dk",                  // spamsources.fabel.dk
	"abuse.spfbl.net",                       // SPFBL.net abuse list
	"dnsbl.spfbl.net",                       // SPFBL.net RBL
	"score.spfbl.net",                       // SPFBL.net Score Service
	"dnswl.spfbl.net",                       // SPFBL.net Whitelist
	"dul.pacifier.net",                      // StopSpam.org dul
	"bl.suomispam.net",                      // Suomispam Blacklist
	"dbl.suomispam.net",                     // Suomispam Domain Blacklist
	"gl.suomispam.net",                      // Suomispam Graylist
	"multi.surbl.org",                       // SURBL multi (Combined SURBL list)
	"srn.surgate.net",                       // SurGATE Reputation Network (SRN)
	"dnsrbl.swinog.ch",                      // Swinog DNSRBL
	"uribl.swinog.ch",                       // Swinog URIBL
	"rbl.tdk.net",                           // TDC's RBL
	"st.technovision.dk",                    // TechnoVision SpamTrap
	"dob.sibl.support-intelligence.net",     // The Day Old Bread List (aka DOB)
	"dbl.tiopan.com",                        // Tiopan Consulting Domain Blacklist
	"bl.tiopan.com",                         // Tiopan Consulting IP Blacklist
	"dnsbl.tornevall.org",                   // TornevallNET DNSBL
	"r.mail-abuse.com",                      // Trend Micro DUL
	"q.mail-abuse.com",                      // Trend Micro QIL
	"rbl2.triumf.ca",                        // TRIUMF.ca DNSBL
	"wbl.triumf.ca",                         // TRIUMF.ca DNSWL
	"truncate.gbudb.net",                    // truncate.gbudb.net
	"dunk.dnsbl.tuxad.de",                   // tuxad dunk.dnsbl
	"hartkore.dnsbl.tuxad.de",               // tuxad hartkore.dnsbl
	"dnsbl-0.uceprotect.net",                // UCEPROTECT Level 0
	"dnsbl-1.uceprotect.net",                // UCEPROTECT Level 1
	"dnsbl-2.uceprotect.net",                // UCEPROTECT Level 2
	"dnsbl-3.uceprotect.net",                // UCEPROTECT Level 3
	"ubl.unsubscore.com",                    // Unsubscribe Blacklist UBL
	"black.uribl.com",                       // URIBL black
	"grey.uribl.com",                        // URIBL grey
	"multi.uribl.com",                       // URIBL multi
	"red.uribl.com",                         // URIBL red
	"white.uribl.com",                       // URIBL white
	"free.v4bl.org",                         // V4BL-FREE/DDNSBL-FREE
	"ip.v4bl.org",                           // V4BL/DDNSBL
	"ips.whitelisted.org",                   // Whitelisted.org
	"blacklist.woody.ch",                    // Woody's SMTP Blacklist IPv4
	"ipv6.blacklist.woody.ch",               // Woody's SMTP Blacklist IPv6
	"uri.blacklist.woody.ch",                // Woody's SMTP Blacklist URIBL
	"bl.blocklist.de",                       // www.blocklist.de
	"dnsbl.zapbl.net",                       // ZapBL DNSRBL
	"rhsbl.zapbl.net",                       // ZapBL RHSBL
}
