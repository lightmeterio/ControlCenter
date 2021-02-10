// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

%%{

machine common;

queueId = xdigit+;

anythingExceptComma = [^,]+;

bracketedEmailLocalPart = [^@>]+;

bracketedEmailDomainPart = [^>]+;

dot = ".";

unknownIP = "unknown";

ipv4 = ([0-9]+dot){3}[0-9]+ | unknownIP;

action setTokBeg { tokBeg = p }

}%%
