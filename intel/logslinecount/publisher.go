// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package logslinecount

import (
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"sync"
)

type counterKey struct {
	process string
	daemon  string
}

type mutablePayloadCounter struct {
	supported   *int
	unsupported *int
}

type Publisher struct {
	sync.Mutex
	counters map[counterKey]mutablePayloadCounter
}

func valuesForNewCounter(payload parser.Payload) (int, int) {
	if payload == nil {
		return 0, 1
	}

	return 1, 0
}

func (p *Publisher) Publish(r postfix.Record) {
	p.Lock()
	defer p.Unlock()

	key := counterKey{process: r.Header.Process, daemon: r.Header.Daemon}

	if v, ok := p.counters[key]; ok {
		if r.Payload == nil {
			*v.unsupported++
			return
		}

		*v.supported++

		return
	}

	supported, unsupported := valuesForNewCounter(r.Payload)

	p.counters[key] = mutablePayloadCounter{supported: &supported, unsupported: &unsupported}
}

func NewPublisher() *Publisher {
	return &Publisher{
		counters: map[counterKey]mutablePayloadCounter{},
	}
}

func flushPublisher(p *Publisher, counters map[counterKey]payloadCounter) {
	p.Lock()
	defer p.Unlock()

	for k, v := range p.counters {
		if (*v.supported)+(*v.unsupported) > 0 {
			counters[k] = payloadCounter{Supported: *v.supported, Unsupported: *v.unsupported}
			*v.supported = 0
			*v.unsupported = 0
		}
	}
}
