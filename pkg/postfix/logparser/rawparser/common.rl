%%{

machine common;

queueId = xdigit+;

anythingExceptComma = [^,]+;

bracketedEmailLocalPart = [^'@']+;

bracketedEmailDomainPart = [^'>']+;

dot = ".";

ipv4 = ([0-9]+dot){3}[0-9]+;

action setTokBeg { tokBeg = p }

}%%
