%%{

machine common;

queueId = xdigit+;

anythingExceptComma = [^,]+;

bracketedEmailLocalPart = [^'@']+;

bracketedEmailDomainPart = [^'>']+;

action setTokBeg { tokBeg = p }

}%%
