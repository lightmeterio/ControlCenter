// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"compress/gzip"
	"errors"
	"fmt"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/rawlogsdb"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"io"
	"net/http"
	"strconv"
	"time"
)

//nolint:structcheck,unused
type fetchLogsHandler struct {
	//nolint:structcheck,unused
	accessor rawlogsdb.Accessor
}

type fetchPagedLogLinesHandler fetchLogsHandler

// @Summary Fetch Log Lines In Time Interval
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param pageSize query integer 0 "Max number of lines to return"
// @Param cursor query integer 0 "Cursor received from the previously fetched page. 0 if first page"
// @Produce json
// @Success 200 {object} rawlogsdb.Content "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/fetchLogLinesInTimeInterval [get]
func (h fetchPagedLogLinesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	pageSize, cursor, err := func() (int, int64, error) {
		pageSizeStr, ok := r.Form["pageSize"]
		if !ok {
			return 0, 0, errors.New(`pageSize argument is missing`)
		}

		cursorStr, ok := r.Form["cursor"]
		if !ok {
			return 0, 0, errors.New(`cursor argument is missing`)
		}

		pageSize, err := strconv.Atoi(pageSizeStr[0])
		if err != nil {
			return 0, 0, errors.New(`Invalid page size`)
		}

		cursor, err := strconv.ParseInt(cursorStr[0], 10, 64)
		if err != nil {
			return 0, 0, errors.New(`Invalid cursor`)
		}

		return pageSize, cursor, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	rows, err := h.accessor.FetchLogsInInterval(r.Context(), interval, pageSize, cursor)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return httputil.WriteJson(w, rows, http.StatusOK)
}

type fetchRawLogLinesToWriterHandler fetchLogsHandler

func formatRawLogsFilename(interval timeutil.TimeInterval) string {
	return fmt.Sprintf(`attachment; filename=logs-%s-%s.log`, interval.From.Format(`20060102`), interval.To.Format(`20060102`))
}

// @Summary Download compressed raw log content from interval
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param format query string gzip "Format of the result. Supported values: gzip, plain"
// @Param disposition query string inline "Use inline to display the response in the browser"
// @Success 200 {object} string "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/fetchRawLogsInTimeInterval [get]
func (h fetchRawLogLinesToWriterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) (err error) {
	interval := httpmiddleware.GetIntervalFromContext(r)

	writer, releaseWriter := func() (io.Writer, func() error) {
		format := r.Form.Get("format")
		disposition := r.Form.Get("disposition")

		switch format {
		case "plain":
			w.Header()["Content-Type"] = []string{"text/plain"}

			w.Header()["Content-Disposition"] = []string{func() string {
				if disposition == "inline" {
					return "inline"
				}

				return formatRawLogsFilename(interval)
			}()}

			return w, func() error { return nil }
		case "gzip":
			fallthrough
		default:
			w.Header()["Content-Type"] = []string{"application/gzip"}
			w.Header()["Content-Disposition"] = []string{formatRawLogsFilename(interval) + `.gz`}
			compressor := gzip.NewWriter(w)

			return compressor, compressor.Close
		}
	}()

	defer errorutil.UpdateErrorFromCall(releaseWriter, &err)

	if err := h.accessor.FetchLogsInIntervalToWriter(r.Context(), interval, writer); err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

type countRawLogLinesHandler fetchLogsHandler

type logLinesCounterResult struct {
	Count int64 `json:"count"`
}

// @Summary Count number of raw log lines in interval
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Success 200 {object} logLinesCounterResult "desc"
// @Failure 422 {string} string "desc"
// @Router /api/v0/countRawLogLinesInTimeInterval [get]
func (h countRawLogLinesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	count, err := h.accessor.CountLogLinesInInterval(r.Context(), interval)
	if err != nil {
		return errorutil.Wrap(err)
	}

	return httputil.WriteJson(w, logLinesCounterResult{Count: count}, http.StatusOK)
}

func HttpRawLogs(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, accessor rawlogsdb.Accessor) {
	authenticated := httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithInterval(timezone))

	mux.Handle("/api/v0/fetchLogLinesInTimeInterval", authenticated.WithEndpoint(fetchPagedLogLinesHandler{accessor}))
	mux.Handle("/api/v0/fetchRawLogsInTimeInterval", authenticated.WithEndpoint(fetchRawLogLinesToWriterHandler{accessor}))
	mux.Handle("/api/v0/countRawLogLinesInTimeInterval", authenticated.WithEndpoint(countRawLogLinesHandler{accessor}))
}
