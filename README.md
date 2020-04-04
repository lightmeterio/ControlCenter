# Lightmeter ControlCenter (work in progress)

![coverage](https://gitlab.com/lightmeter/controlcenter/badges/master/coverage.svg)
![pipeline](https://gitlab.com/lightmeter/controlcenter/badges/master/pipeline.svg)

Welcome to Lightmeter, the Open Source email deliverability monitoring system.

## Supported Mail Transfer Agents

Currently Postfix MTA is supported. Future support for additional MTAs is planned.

## Status

This is a next generation rewrite of the previous [prototype](https://gitlab.com/lightmeter/prototype), and is currently work in progress.

## Build from source code

You'll need the Go compiler installed. Check http://golang.org for more information. The Go version we are currently using is 1.14.1.

To build Lightmeter during development, execute:

```
./build.sh dev
```

And for the final release, execute:
```
./build.sh release

```

Extra build arguments can be passed to the build.sh script. For instance, to create a static linked version, execute:
```
./build.sh release -a -ldflags "-linkmode external -extldflags '-static' -s -w"
```

That will download all the dependencies and build a file called `lightmeter`,
which you can simply copy to your Postfix server and use it as described in the `Usage` section.

### Cross compilation

To cross compile, for instance, to Windows, execute, adapting to your environment:

```
CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 ./build.sh release
```

Which will create a file called `lightmeter.exe`.

It's good to remember that we probably won't ever support Windows, but that does not mean you cannot use it there ;-)

## Usage

- Run `lightmeter -help` to show a list of all available commands
- Following compilation of Lightmeter Control Center you should run the binary `lightmeter` to read logs and launch a local webserver which allows viewing Lightmeter Control Center via a web ui in a browser on the same network on port 8080, eg. [http://localhost:8080/](http://localhost:8080/).
    - Logfiles provided via `watch` will be monitored for changes and the web ui automatically updated. An SQLite database is used in the backend for storing processed log data.
- Specify which mail logs to watch using the command line argument `lightmeter -watch [path/to/logfile.log]`. This argument can be specified multiple times to read from multiple files.
- To supply logs via stdin instead of logfile location, use the command line argument `-stdin` like `lightmeter -stdin < [log-data]`.
- Postfix logs don't contain a year as part of the date of each line, so the year is assumed to be this year. To override this and specify a year manually, use the `-what_year_is_it` flag like `-what_year_is_it 2018` 
