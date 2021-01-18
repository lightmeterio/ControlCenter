%%{

machine common;

queueId = xdigit+;

anythingExceptComma = [^,]+;

bracketedEmailLocalPart = [^@]+;

bracketedEmailDomainPart = [^>]+;

dot = ".";

unknownIP = "unknown";

ipv4 = ([0-9]+dot){3}[0-9]+ | unknownIP;

action setTokBeg { tokBeg = p }

}%%
