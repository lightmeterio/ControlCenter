// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dbconn

import (
	"database/sql"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"sync"
)

type Mode int

const (
	ROMode Mode = 0
	RWMode Mode = 1
)

type CounterKey struct {
	Filename string
	Mode     Mode
}

var (
	connCounters      = map[CounterKey]*int{}
	connCountersMutex sync.Mutex
)

func CountDetails() map[CounterKey]int {
	m := map[CounterKey]int{}

	connCountersMutex.Lock()

	defer connCountersMutex.Unlock()

	for k, v := range connCounters {
		if *v != 0 {
			m[k] = *v
		}
	}

	return m
}

func incConnCounter(filename string, mode Mode, inc int) {
	connCountersMutex.Lock()

	defer connCountersMutex.Unlock()

	counter := func() *int {
		key := CounterKey{Filename: filename, Mode: mode}
		if c, ok := connCounters[key]; ok {
			return c
		}

		c := 0

		connCounters[key] = &c

		return &c
	}()

	*counter += inc
}

func CountOpenConnections() int {
	connCountersMutex.Lock()

	defer connCountersMutex.Unlock()

	c := 0

	for _, v := range connCounters {
		c += *v
	}

	return c
}

type connCloser struct {
	filename string
	mode     Mode
	db       *sql.DB
}

func newConnCloser(filename string, mode Mode, db *sql.DB) *connCloser {
	incConnCounter(filename, mode, 1)

	log.Debug().Msgf("Opening (%d) database file %s, count: %d", mode, filename, CountOpenConnections())

	return &connCloser{
		filename: filename,
		mode:     mode,
		db:       db,
	}
}

func (c *connCloser) Close() error {
	incConnCounter(c.filename, c.mode, -1)

	log.Debug().Msgf("Closing (%d) database file %s, count: %d", c.mode, c.filename, CountOpenConnections())

	if err := c.db.Close(); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
