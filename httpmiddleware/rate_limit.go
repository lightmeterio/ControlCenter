// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package httpmiddleware

import (
	"errors"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/pkg/ctxlogger"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type queryCount struct {
	startTs time.Time
	count   int64
}

// Data structures to define rate-limiting behaviour
type RestrictAction func() int

func BlockQuery() int {
	return http.StatusTooManyRequests
}

type rateLimit struct {
	numberOfTries int64 // after this number of accesses, the corresponding action will be taken
	action        RestrictAction
}

type rateLimitsByEndPoint struct {
	timeFrame time.Duration // time after which an IP will be free to access endpoint again
	limits    []rateLimit
}

func RequestWithRateLimit(timeFrame time.Duration, numberOfTries int64, isBehindAReverseProxy bool, action RestrictAction) Middleware {
	return requestWithRateLimitAndWithCustomClock(&timeutil.RealClock{}, timeFrame, numberOfTries, isBehindAReverseProxy, action)
}

func requestRemoteAddr(remoteAddr string) string {
	index := strings.LastIndex(remoteAddr, ":")
	if index == -1 {
		return remoteAddr
	}

	return remoteAddr[:index]
}

func remoteAddr(requestRemoteAddr string, header http.Header, isBehindAReverseProxy bool) string {
	if !isBehindAReverseProxy {
		return requestRemoteAddr
	}

	// Get original IP if behind a proxy - apache, traefik, probably nginx - should mostly be the case
	if originAddr, ok := header["X-Forwarded-For"]; ok {
		return originAddr[0]
	}

	log.Debug().Msgf("Could not obtain the client IP address from the headers. Here they are: %#v", header)

	return requestRemoteAddr
}

func requestWithRateLimitAndWithCustomClock(clock timeutil.Clock, timeFrame time.Duration, numberOfTries int64, isBehindAReverseProxy bool, action RestrictAction) Middleware {
	var (
		rateLimits = rateLimitsByEndPoint{
			timeFrame: timeFrame,
			limits: []rateLimit{
				{
					numberOfTries: numberOfTries,
					action:        action,
				},
			},
		}

		counters = map[string]*queryCount{}
		mutex    sync.Mutex
	)

	return func(h CustomHTTPHandler) CustomHTTPHandler {
		return CustomHTTPHandler(func(w http.ResponseWriter, r *http.Request) error {
			remoteAddr := remoteAddr(requestRemoteAddr(r.RemoteAddr), r.Header, isBehindAReverseProxy)

			// NOTE: we wrap this code in a function not to block the mutex for very long
			err := func() error {
				mutex.Lock()
				defer mutex.Unlock()

				now := clock.Now()

				counterForIP, ok := counters[remoteAddr]

				if !ok {
					counterForIP = &queryCount{startTs: now, count: 0}
					counters[remoteAddr] = counterForIP
				}

				elapsedTime := now.Sub(counterForIP.startTs)

				if elapsedTime > rateLimits.timeFrame {
					// Timeframe is over, start a new counter
					counterForIP = &queryCount{startTs: now, count: 0}
					counters[remoteAddr] = counterForIP
				}

				counterForIP.count++

				httpStatus := applyRateLimitsendPointRateLimits(&rateLimits, counterForIP)

				// cleanup: delete every other ip/endpoints counters whose timeframe has elapsed
				for ip, counter := range counters {
					// skip counter currently handled
					if counter == counterForIP {
						continue
					}

					elapsedTime := now.Sub(counter.startTs)

					if elapsedTime > rateLimits.timeFrame {
						delete(counters, ip)
					}
				}

				if httpStatus == http.StatusOK {
					return nil
				}

				err := httperror.NewHTTPStatusCodeError(httpStatus, errors.New("Query blocked by rate limiter"))
				ctxlogger.LogErrorf(r.Context(), err, "Query blocked by rate limiter")

				response := struct {
					Error string `json:"error"`
				}{
					Error: "Blocked for exceeding rate-limit, please try again later",
				}

				return httputil.WriteJson(w, response, httpStatus)
			}()

			if err != nil {
				return err
			}

			return h.ServeHTTP(w, r)
		})
	}
}

func applyRateLimitsendPointRateLimits(endPointRateLimits *rateLimitsByEndPoint, endpointCount *queryCount) int {
	for _, limit := range endPointRateLimits.limits {
		if endpointCount.count > limit.numberOfTries {
			if httpStatus := limit.action(); httpStatus != http.StatusOK {
				return httpStatus
			}
		}
	}

	return http.StatusOK
}
