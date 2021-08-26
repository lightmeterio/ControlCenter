<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
SPDX-License-Identifier: AGPL-3.0-only
-->

# Lightmeter Agent

This is an experiment for a daemon that runs together with Postfix and is responsible for communicating with it.

It will also be able to work as a bridge to other software, as Dovecot.

So far the only thing implemented is obtaining Postfix configuration via `postconf`.

## Running it with docker:

First of, start Postfix, via docker:

```sh
> docker run -it --rm --name my_postfix alpine:latest
# Inside docker:
> apk update && apk add postfix rsyslog && rsyslogd && postfix start && tail -f /var/log/mail.log
```

Then in another terminal, run the test program that instructs Postfix to block a list of IP addresses
of connecting to the server.

```sh
export POSTFIX_CONTAINER=my_postfix
cd example
go run ./block-ips.go 123.456.789.123 [more ips here, separated by space...]
```

## Running it without Docker

If you don't export the environment variable `POSTFIX_CONTAINER`, the agent will consider that Postfix is running
in the same system (therefore having direct access to Postfix), and docker won't be required.
