#!/usr/bin/env python3

# Clean up postfix logs. Receive logs via stdin and writes cleaned version to stdout
# Simple like that.
# It's slow as hell, due lots of intermediate strings created on each substitution
# and the recursive calls.
# But it does the job for now.
# TODO: implement some unit testing

def replace_ip_v4(s, c, spans):
    return [s[:spans(0)[0]], "11.22.33.44", s[spans(0)[1]:]]

def replace_email(s, c, spans):
    import hashlib

    local_part = s[spans(1)[0]:spans(1)[1]]
    domain_part = s[spans(2)[0]:spans(2)[1]]

    hashed_local_part = hashlib.sha1(local_part.encode()).hexdigest()[:len(local_part)]
    hashed_domain_part = hashlib.sha1(domain_part.encode()).hexdigest()[:len(domain_part)]

    return [s[:spans(0)[0]], "h-", hashed_local_part, "@h-", hashed_domain_part, ".com", s[spans(0)[1]:]]

def replace_domain(s, c, spans):
    def hashed_value():
        import hashlib
        domain = s[spans(1)[0]:spans(1)[1]]

        if domain.startswith('h-'): # has already been hashed by the email replacer
            return [domain]

        return ['h-', hashlib.sha1(domain.encode()).hexdigest()[:len(domain)]]

    return [s[:spans(1)[0]]] + hashed_value() + [s[spans(1)[1]:]]

def replace_unix_file_path(s, c, spans):
    import hashlib
    path = s[spans(2)[0]:spans(2)[1]]
    return [s[:spans(2)[0]], '/h-', hashlib.sha1(path.encode()).hexdigest(), '/', s[spans(3)[1]:]]

# A pattern is composed by a tuple consisting on a regex as first element
# and a function(string, re.Pattern, Span) -> list<string>
# Where Span = function(int) -> tuple(int, int) (please check the documentation for re.Match.span)
# Uff, that's it (I am not yet used to do type annotations in Python :-()
patterns = [
    (r'(\s)((/[\.\w_-]+)+)([^[])?', replace_unix_file_path),
    (r'\b([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})\b', replace_ip_v4),
    (r'([-\w_\.]+)@([-\w_\.]+)', replace_email), # unquoted emails
    (r'"([-\w_\.]+)"@([-\w_\.]+)', replace_email), # quoted emails, like in to=<"I have spaces"@domain.de>
    # There's no official regexp for validating domains/hostnames. This is the best I could came up with so far :-(
    # FIXME: It's very heavy and I noticed that more than half of the time (on my test files) is spent inside this
    # regular expression!
    # It'd be nice to optimize it by something more lightweight
    (r'(((([a-zA-Z_][\-a-zA-Z-0-9_]*)|([0-9][\-a-zA-Z_][\-a-zA-Z0-9_]*))(\.(([a-zA-Z_][\-a-zA-Z0-9_]*)|([0-9][\-a-zA-Z_][\-a-zA-Z0-9_]*))))+)[^\.]', replace_domain),
   ]

def compile_regex(p):
    import sys
    import re

    try:
        return re.compile(p)
    except:
        print(f'Failed to build regex {p}', file=sys.stderr)
        sys.exit(1)

compiled_patterns = [(compile_regex(p), f) for (p, f) in patterns]

def clean_pattern(level, s, c, r):
    import re
    m = re.search(c, s)

    if m is None:
        # do not bother to build a list if there's nothing to be replaced
        # and no recursive call has been yet made
        return [s] if level > 0 else None

    spans = m.span

    return r(s[:spans(0)[1]], c, spans) + clean_pattern(level + 1, s[spans(0)[1]:], c, r)

def clean_line(line):
    stripped = line.rstrip()

    for p in compiled_patterns:
        replaced_or_none = clean_pattern(0, stripped, p[0], p[1])
        if replaced_or_none is not None:
            stripped = ''.join(replaced_or_none)

    return stripped

def main():
    import sys
    from multiprocessing import Pool, cpu_count

    p = Pool(cpu_count() + 1)

    for c in p.map(clean_line, sys.stdin):
        print(c)

if __name__ == '__main__':
    main()
