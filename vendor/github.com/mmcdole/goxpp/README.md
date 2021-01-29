# goxpp

[![Build Status](https://travis-ci.org/mmcdole/goxpp.svg?branch=master)](https://travis-ci.org/mmcdole/goxpp) [![Coverage Status](https://coveralls.io/repos/github/mmcdole/goxpp/badge.svg?branch=master)](https://coveralls.io/github/mmcdole/goxpp?branch=master) [![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org)
[![GoDoc](https://godoc.org/github.com/mmcdole/goxpp?status.svg)](https://godoc.org/github.com/mmcdole/goxpp)

The `goxpp` library is an XML parser library that is loosely based on the [Java XMLPullParser](http://www.xmlpull.org/v1/download/unpacked/doc/quick_intro.html).  This library allows you to easily parse arbitrary XML content using a pull parser.  You can think of `goxpp` as a lightweight wrapper around Go's XML `Decoder` that provides a set of functions that make it easier to parse XML content than using the raw decoder itself.

## Overview

To begin parsing a XML document using `goxpp` you must pass it an `io.Reader` object for your document:

```go
file, err := os.Open("path/file.xml")
parser := xpp.NewXMLPullParser(file, false, charset.NewReader)
```

The `goxpp` library decodes documents into a series of token objects:

| Token Name                       |
|----------------------------------|
| 	StartDocument                  |
| 	EndDocument                    |
| 	StartTag                       |
| 	EndTag                         |
| 	Text                           |
| 	Comment                        |
| 	ProcessingInstruction          |
| 	Directive                      |
| 	IgnorableWhitespace            |

You will always start at the `StartDocument` token and can use the following functions to walk through a document:

| Function Name                    | Description                           |
|----------------------------------|---------------------------------------|
| 	 Next()                        | Advance to the next `Text`, `StartTag`, `EndTag`, `EndDocument` token.<br>Note: skips `Comment`, `Directive` and `ProcessingInstruction` |
| 	NextToken()                    | Advance to the next token regardless of type.                                                                |
| 	NextText()                     | Advance to the next `Text` token.                                                                |
| 	Skip()                         | Skip the next token.   |
| 	DecodeElement(v interface{})   | Decode an entire element from the current tag into a struct.<br>Note: must be at a `StartTag` token |



This project is licensed under the [MIT License](https://raw.githubusercontent.com/mmcdole/goxpp/master/LICENSE)

