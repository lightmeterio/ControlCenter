// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// following regexp helps remove the :port part of RemoteAddrs
// (a simple split on ':' is not enough since the IP can be ipv6 and contain this character)
var remoteAddrSplit = regexp.MustCompile(`:\d+$`)

// Data structures for keeping count of accesses by IP and URL
type ipAddr string
type endpoint string

type queryCount struct {
	startTs time.Time
	count   int64
}

type queryCountByEndpoint map[endpoint]*queryCount
type queryCounter map[ipAddr]*queryCountByEndpoint

var qCounter = queryCounter{}
var muLockQueryCounter sync.Mutex

// Data structures to define rate-limiting behaviour
type restrictAction func() int

func blockQuery() int {
	return http.StatusTooManyRequests
}

type rateLimit struct {
	numberOfTries int64 // after this number of accesses, the corresponding action will be taken
	action        restrictAction
}

type rateLimitsByEndPoint struct {
	timeFrame time.Duration // time after which an IP will be free to access endpoint again
	limits    []rateLimit
}

type rateLimits map[endpoint]rateLimitsByEndPoint

var commonRateLimits = rateLimits{
	endpoint("/login"): rateLimitsByEndPoint{
		timeFrame: time.Minute * time.Duration(5),
		limits: []rateLimit{
			{
				numberOfTries: 20,
				action:        blockQuery,
			},
		},
	},
	endpoint("/api/v0/checkMessageDeliveryStatus"): rateLimitsByEndPoint{
		timeFrame: time.Minute * time.Duration(10),
		limits: []rateLimit{
			{
				numberOfTries: 20,
				action:        blockQuery,
			},
		},
	},
}

func GetMaxNumberOfTriesForEndpoint(url string) int64 {
	return commonRateLimits[endpoint(url)].limits[0].numberOfTries
}

func addQuery(r *http.Request) int {
	urlParts := strings.Split(r.URL.String(), "?")
	url := endpoint(urlParts[0])

	endPointRateLimits := applicableRateLimits(url)
	if endPointRateLimits == nil {
		return http.StatusOK
	}

	userIP := remoteAddrSplit.Split(r.RemoteAddr, -1)

	remoteAddr := ipAddr(userIP[0])

	// Get original IP if behind a proxy - apache, traefik, probably nginx - should mostly be the case
	originAddr, ok := r.Header["X-Forwarded-For"]

	if ok && (remoteAddr == "127.0.0.1" || remoteAddr == "[::1]") {
		remoteAddr = ipAddr(originAddr[0])
	}

	muLockQueryCounter.Lock()
	defer muLockQueryCounter.Unlock()

	ipQueryCount, ok := qCounter[remoteAddr]

	if !ok {
		qCounter[remoteAddr] = &queryCountByEndpoint{url: &queryCount{time.Now(), 0}}
		ipQueryCount = qCounter[remoteAddr]
	}

	endpointCount, endpointCounterValid := (*ipQueryCount)[url]

	// we had started counting queries for this ip+url, check when, and if timeframe is over
	if endpointCounterValid {
		elapsedTime := time.Since(endpointCount.startTs)

		if elapsedTime > endPointRateLimits.timeFrame {
			// Timeframe is over, start a new counter
			endpointCounterValid = false
		}
	}

	if !endpointCounterValid {
		(*ipQueryCount)[url] = &queryCount{time.Now(), 0}
		endpointCount = (*ipQueryCount)[url]
	}

	endpointCount.count++

	httpStatus := applyRateLimits(url, endpointCount)

	// cleanup: delete other ip/endpoints counters whose timeframe has elapsed
	for ip, ipQueryCount := range qCounter {
		for url, endpointCount := range *ipQueryCount {
			elapsedTime := time.Since(endpointCount.startTs)

			if elapsedTime > applicableRateLimits(url).timeFrame {
				delete(*ipQueryCount, url)
			}
		}

		if len(*ipQueryCount) == 0 {
			delete(qCounter, ip)
		}
	}

	return httpStatus
}

func applicableRateLimits(url endpoint) *rateLimitsByEndPoint {
	endPointRateLimits, exists := commonRateLimits[url]

	if !exists {
		return nil
	}

	return &endPointRateLimits
}

func applyRateLimits(url endpoint, endpointCount *queryCount) int {
	endPointRateLimits := applicableRateLimits(url)

	if endPointRateLimits == nil {
		return http.StatusOK
	}

	for _, limit := range endPointRateLimits.limits {
		if endpointCount.count > limit.numberOfTries {
			if httpStatus := limit.action(); httpStatus != http.StatusOK {
				return httpStatus
			}
		}
	}

	return http.StatusOK
}
