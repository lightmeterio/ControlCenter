# Lightmeter ControlCenter (work in progress)

Welcome to Lightmeter, the Open Source email deliverability monitoring system.

## Supported Mail Transfer Agents

Currently Postfix MTA is supported. Future support for additional MTAs is planned.

## Status

This is a next generation rewrite of the previous [prototype](https://gitlab.com/lightmeter/prototype), and is currently work in progress.

## Usage

- Following compilation of Lightmeter Control Center you should run the binary `read_postfix_logs` to read logs and launch a local webserver which allows viewing Lightmeter Control Center via a web ui in a browser on the same network on port 8080, eg. [http://localhost:8080/](http://localhost:8080/).
    - Logfiles provided via `watch` will be monitored for changes and the web ui automatically updated. An SQLite database is used in the backend for storing processed log data.
- Specify which mail logs to watch using the command line argument `read_postfix_logs -watch [path/to/logfile.log]`. This argument can be specified multiple times to read from multiple files.
- To supply logs via stdin instead of logfile location, use the command line argument `-stdin` like `read_postfix_logs -stdin < [log-data]`.
