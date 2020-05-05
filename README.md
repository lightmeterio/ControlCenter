# Lightmeter ControlCenter (work in progress)

[![pipeline status](https://gitlab.com/lightmeter/controlcenter/badges/master/pipeline.svg)](https://gitlab.com/lightmeter/controlcenter/-/commits/master)
[![coverage report](https://gitlab.com/lightmeter/controlcenter/badges/master/coverage.svg)](https://gitlab.com/lightmeter/controlcenter/-/commits/master)
[![report_card](https://goreportcard.com/badge/gitlab.com/lightmeter/controlcenter)](https://goreportcard.com/report/gitlab.com/lightmeter/controlcenter)

Welcome to Lightmeter, the Open Source email deliverability monitoring system.

## Supported Mail Transfer Agents

Currently Postfix MTA is supported. Future support for additional MTAs is planned.

## Status

This is a next generation rewrite of the previous [prototype](https://gitlab.com/lightmeter/prototype), and is currently work in progress.

## Build from source code

You'll need the Go compiler installed. Check http://golang.org for more information. The Go version we are currently using is 1.14.1.

To build Lightmeter during development, execute:

```
make dev
```

And for the final release, execute:
```
make release

```

And to create a static linked (please use carefully) version, execute:
```
make static_release
```

That will download all the dependencies and build a file called `lightmeter`,
which you can simply copy to your Postfix server and use it as described in the `Usage` section.

### Cross compilation

To compile to Windows, using Linux as host (requires cross compiler):

```
make windows_release
```

Which will create a file called `lightmeter.exe`.

It's good to remember that we probably won't ever support Windows, but that does not mean you cannot use it there ;-)

## Usage

- Run `lightmeter -help` to show a list of all available commands
- Following compilation (or download) of Lightmeter Control Center you should run the binary `lightmeter` to read logs and launch a local webserver, which allows viewing Lightmeter Control Center via a Web UI in a browser on the same network on port 8080, eg. [http://localhost:8080/](http://localhost:8080/).
    - Logfiles provided using the `-watch` argument will be monitored for changes and the Web UI automatically updated. An SQLite database is used in the backend for storing processed log data. Note that `-watch` only looks for new changes from the last recorded time of the previous import; therefore it does not scan the entire contents of the specified logfile if it has previously been imported or watched.
- Specify which mail logs to watch using the command line argument `lightmeter -watch [path/to/logfile.log]`. This argument can be specified multiple times to read from multiple files.
- To supply logs via stdin instead of logfile location, use the command line argument `-stdin` like `lightmeter -stdin < [log-data]`.
- Mailserver data is stored in separate workspaces so that different servers can be monitored separately. See `-help` for more details on managing these.
- Postfix logs don't contain a year as part of the date of each line, so the year for processed logs is assumed to be this year. To override this and specify a year manually, use the `-what_year_is_it` flag like `-what_year_is_it 2018` 

### API

Lightmeter ships with a simple REST API designed for user interfaces. It is used by the Web UI. 

Swagger-based API documentation and experimentation pages are generated automatically on development builds. Access them via `http://lightmeter-address:8080/api`, eg. [http://localhost:8080/api](http://localhost:8080/api).
