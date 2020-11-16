# Postfix Log Parser Go Library

This is an eternal work in progress Postfix Log Parser used by Lightmeter Control Center.

It's very imcomplete as it for now parses only the log types that we use on Lightmeter.

If you have interest on improving it, please send us a PR and check the Lightmeter Contributor agreement in the file CLA.

## Building

You'll need [ragel](http://www.colm.net/open-source/ragel/) to generate code from some state machines with go generate.

Then run:

```sh
$ go generate ./rawparser
```

## License

For licencing information, please check the file LICENSE.

Lightmeter Team
https://lightmeter.io
