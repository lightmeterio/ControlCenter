# Lightmeter ControlCenter (work in progress)

[![pipeline status](https://gitlab.com/lightmeter/controlcenter/badges/master/pipeline.svg)](https://gitlab.com/lightmeter/controlcenter/-/commits/master)
[![coverage report](https://gitlab.com/lightmeter/controlcenter/badges/master/coverage.svg)](https://gitlab.com/lightmeter/controlcenter/-/commits/master)
[![report_card](https://goreportcard.com/badge/gitlab.com/lightmeter/controlcenter)](https://goreportcard.com/report/gitlab.com/lightmeter/controlcenter)
[![scale_index](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=sqale_index)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![bugs](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=bugs)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![code_smells](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=code_smells)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![coverage](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=coverage)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![duplicated_lines_density](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=duplicated_lines_density)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![ncloc](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=ncloc)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![sqale_rating](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![alert_status](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=alert_status)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![reliability_rating](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![security_rating](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=security_rating)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![sqale_index](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=sqale_index)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)
[![vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=lightmeter_controlcenter&metric=vulnerabilities)](https://sonarcloud.io/dashboard?id=lightmeter_controlcenter)

Welcome to Lightmeter Control Center, the Open Source mailtech monitoring and management application.

![Lightmeter Control Center 0.0.2 screenshot](https://lightmeter.io/wp-content/uploads/2020/06/Selection_816.png "Lightmeter Control Center 0.0.2 screenshot")

## Quickstart

1. Install Lightmeter Control Center as you prefer, then run it using `./lightmeter -watch_dir /path/to/mail/log/dir`
1. Open `http://localhost:8080/` to see the interface
1. If necessary change the date range to see graphs for the period of the logs you just imported

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
- Following compilation (or download) of Lightmeter Control Center you should run the binary `lightmeter` to read logs and launch a local webserver, which allows viewing Lightmeter Control Center via a Web UI in a browser on the same network on port 8080, eg. [http://localhost:8080/](http://localhost:8080/). You can use `-listen ":9999"` for instance to use a different port or network interface, in this case all interfaces on port 9999.
    - Logfiles provided using the `-watch` argument will be monitored for changes and the Web UI automatically updated. An SQLite database is used in the backend for storing processed log data. Note that `-watch` only looks for new changes from the last recorded time of the previous import; therefore it does not scan the entire contents of the specified logfile if it has previously been imported or watched.
- Specify which mail logs to watch using the command line argument `lightmeter -watch [path/to/logfile.log]`. This argument can be specified multiple times to read from multiple files.
- To supply logs via stdin instead of logfile location, use the command line argument `-stdin` like `lightmeter -stdin < [log-data]`.
- Mailserver data is stored in separate workspaces so that different servers can be monitored separately. See `-help` for more details on managing these.
- Postfix logs don't contain a year as part of the date of each line, so the year for processed logs is assumed to be this year. To override this and specify a year manually, use the `-what_year_is_it` flag like `-what_year_is_it 2018` 
- Lightmeter can also "watch" a directory with postfix logs managed by logrotate, importing existing files
(even if compressed with gzip) and waiting new log files that happen after such import.
To use it, start lightmeter with the argument `-watch_dir /path/to/dir`, which is likely to be `/var/log/mail`.
Lightmeter won't import such logs again if they have already been imported, in case of a process restart.
Currently the following patterns for log files are "watched":
  - mail.log
  - mail.info
  - mail.warn
  - mail.err

The importing process will take a long time, depending on how many files you have and how big they are.

It's important not to use `-watch_dir` with other ways of obtaining logs, and future versions of Lightmeter will disable such behaviour.

In case you are having an error like this:

```
2020/05/29 13:45:05 Missing file mail.log . Instead, found:  /var/log/mail/mail.log.2.gz

```

This means you should have a file mail.log, which means you should check your Postfix installation and ensure it's emitting logs properly.

### API

Lightmeter ships with a simple REST API designed for user interfaces. It is used by the Web UI. 

Swagger-based API documentation and experimentation pages are generated automatically on development builds. Access them via `http://lightmeter-address:8080/api`, eg. [http://localhost:8080/api](http://localhost:8080/api).

