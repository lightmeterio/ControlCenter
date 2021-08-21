// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package parser

import (
	"bufio"
	"bytes"
	"errors"
	"regexp"
)

// FIXME: The value for a line is not regular, therefore cannot be fully parsed using Regexp!!!!
var lineRegexp = regexp.MustCompile(`([\w_]+)\s=(\s(([^$].*)|\$(.+)))?`)

type entry struct {
	parser *Parser

	key           string
	rawValue      string
	absoluteValue string
	variableValue string
}

type Parser struct {
	entries map[string]entry
}

var ErrParsing = errors.New(`Parsing error`)

func Parse(content []byte) (*Parser, error) {
	buffer := bytes.NewBuffer(content)
	scanner := bufio.NewScanner(buffer)

	p := &Parser{entries: map[string]entry{}}

	for scanner.Scan() {
		line := scanner.Bytes()
		matches := lineRegexp.FindSubmatch(line)

		if len(matches) == 0 {
			// TODO: inform line?
			return nil, ErrParsing
		}

		key := string(matches[1])
		rawValue := string(matches[3])
		absoluteValue := string(matches[4])
		variableValue := string(matches[5])

		p.entries[key] = entry{parser: p, key: key, absoluteValue: absoluteValue, variableValue: variableValue, rawValue: rawValue}
	}

	return p, nil
}

var ErrKeyNotFound = errors.New(`Key not found`)

// TODO: Meh! It turns out that `postfix -x` already resolves all the settings,
// leaving with no needs for a custom variable parser!
func (p *Parser) Resolve(key string) (string, error) {
	if v, ok := p.entries[key]; ok {
		return v.Resolve(), nil
	}

	return "", ErrKeyNotFound
}

func (p *Parser) Value(key string) (string, error) {
	if v, ok := p.entries[key]; ok {
		return v.Value(), nil
	}

	return "", ErrKeyNotFound
}

func (e *entry) Value() string {
	return e.rawValue
}

func (e *entry) Resolve() string {
	if len(e.absoluteValue) > 0 {
		return e.absoluteValue
	}

	otherEntry, ok := e.parser.entries[e.variableValue]
	if !ok {
		return ""
	}

	return otherEntry.Resolve()
}
