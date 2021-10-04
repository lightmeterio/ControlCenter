// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

%%{

machine common;

# this does not really match the definition at http://www.postfix.org/postconf.5.html#enable_long_queue_ids,
# but is close enough to work for us.
longQueueId = [0-9a-zA-Z]{12,};
shortQueueId = [0-9A-F]{6,};
queueId = shortQueueId | longQueueId;

anythingExceptComma = [^,]+;

bracketedEmailLocalPart = [^@>]+;

bracketedEmailDomainPart = [^>]+;

dot = ".";

unknownIP = "unknown";

ipv4 = ([0-9]+dot){3}[0-9]+ | unknownIP;

squareBracketedValue = [^\]]+;

action setTokBeg { tokBeg = p }

}%%
