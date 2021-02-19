// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package timeutil

import (
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now()
}

func (RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

type FakeClock struct {
	// Locking used just to prevent the race detector of triggering errors during tests
	sync.Mutex
	time.Time
}

func (t *FakeClock) Now() time.Time {
	t.Lock()
	defer t.Unlock()

	return t.Time
}

func (t *FakeClock) Sleep(d time.Duration) {
	t.Lock()
	defer t.Unlock()
	t.Time = t.Time.Add(d)
}
