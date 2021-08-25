<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
SPDX-License-Identifier: AGPL-3.0-only
-->

# Lightmeter Agent

This is an experiment for a daemon that runs together with Postfix and is responsible for communicating with it.

It will also be able to work as a bridge to other software, as Dovecot.

So far the only thing implemented is obtaining Postfix configuration via `postconf`.

# Running it:

First of, start Postfix, via docker:

```sh
> docker run -it --rm --name my_postfix alpine:latest
# Inside docker:
> apk update && apk add postfix
```

Then in another terminal, run the agent:
```sh
export POSTFIX_CONTAINER=my_postfix
cd example
go run ./main.go
```
