// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build release
// +build release

package newsletter

import (
	"net/http"
	"time"
)

func NewSubscriber(url string) Subscriber {
	// Client-side timeouts to prevent leaking resources or getting stuck.
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	return &HTTPSubscriber{URL: url, HTTPClient: httpClient}
}
