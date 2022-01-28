// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"errors"
	"gitlab.com/lightmeter/controlcenter/httpauth/auth"
	"gitlab.com/lightmeter/controlcenter/httpmiddleware"
	"gitlab.com/lightmeter/controlcenter/insights"
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/pkg/httperror"
	"gitlab.com/lightmeter/controlcenter/recommendation"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"gitlab.com/lightmeter/controlcenter/util/httputil"
	"gitlab.com/lightmeter/controlcenter/util/timeutil"
	"net/http"
	"strconv"
	"time"
)

type fetchInsightsHandler struct {
	f                          core.Fetcher
	recommendationURLContainer recommendation.URLContainer
}

// @Summary Fetch Insights
// @Produce json
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param filter query string false "Filter by. Possible values: 'category'" Enums{"category"}
// @Param order query string true "Order by. Possible values: 'creationAsc', 'creationDesc'" Enums{"creationAsc", "creationDesc"}
// @Param entries query int false "Maximum number of insights to fetch"
// @Param category query string false "If filter by category, the category name. Possible values: 'local', 'comparative', 'news', 'intel'" Enums{"local", "comparative", "news", "intel"}
// @Success 200 {object} fetchedInsight
// @Failure 422 {string} string "desc"
// @Router /api/v0/fetchInsights [get]
func (h fetchInsightsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	interval := httpmiddleware.GetIntervalFromContext(r)

	filter := core.BuildFilterByName(r.Form.Get("filter"))
	order := core.BuildOrderByName(r.Form.Get("order"))
	category := core.BuildCategoryByName(r.Form.Get("category"))

	entries, err := func() (int, error) {
		s := r.Form.Get("entries")

		if len(s) == 0 {
			return 0, nil
		}

		return strconv.Atoi(s)
	}()

	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errorutil.Wrap(err, "Invalid entries query value"))
	}

	if entries < 0 {
		return httperror.NewHTTPStatusCodeError(http.StatusBadRequest, errors.New("Invalid entries query value: negative value"))
	}

	fetchedInsights, err := h.f.FetchInsights(r.Context(), core.FetchOptions{
		Interval:   interval,
		Category:   category,
		FilterBy:   filter,
		OrderBy:    order,
		MaxEntries: entries,
	}, timeutil.RealClock{})

	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusInternalServerError, errorutil.Wrap(err))
	}

	insights := make(fetchInsightsResult, 0, entries)

	for _, fi := range fetchedInsights {
		i := fetchedInsight{
			ID:            fi.ID(),
			Time:          fi.Time(),
			Rating:        fi.Rating().String(),
			Category:      fi.Category().String(),
			ContentType:   fi.ContentType(),
			Content:       fi.Content(),
			UserRating:    fi.UserRating(),
			UserRatingOld: fi.UserRatingOld(),
		}

		if recommendationHelpLinkProvider, ok := fi.Content().(core.RecommendationHelpLinkProvider); ok {
			i.HelpLink = recommendationHelpLinkProvider.HelpLink(h.recommendationURLContainer)
		}

		insights = append(insights, i)
	}

	return httputil.WriteJson(w, insights, http.StatusOK)
}

type fetchedInsight struct {
	ID            int         `json:"id"`
	Time          time.Time   `json:"time"`
	Rating        string      `json:"rating"`
	Category      string      `json:"category"`
	ContentType   string      `json:"content_type"`
	Content       interface{} `json:"content"`
	HelpLink      string      `json:"help_link,omitempty"`
	UserRating    *int        `json:"user_rating"`
	UserRatingOld bool        `json:"user_rating_old"`
}

type fetchInsightsResult []fetchedInsight

type rateInsightHandler struct {
	e *insights.Engine
}

// @Summary Rate insight usefulness
// @Produce json
// @Param type   query string true "Insight content type to rate"
// @Param rating query string true "A rating among 0 (not useful), 1 (not so useful) and 2 (useful)"
// @Success 200 {string} string ""
// @Failure 422 {string} string "desc"
// @Router /api/v0/rateInsight [post]
func (h rateInsightHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.ParseForm() != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, errors.New("Wrong Input"))
	}

	rating, err := strconv.Atoi(r.Form.Get("rating"))
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	if err = h.e.RateInsight(r.Form.Get("type"), uint(rating), timeutil.RealClock{}); err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	return nil
}

type archiveInsightHandler struct {
	e *insights.Engine
}

// @Summary Archive an insight
// @Produce json
// @Param id query integer 0 "Insight id"
// @Success 200 {string} string "Insight was archived"
// @Failure 422 {string} string "Wrong parameter"
// @Router /api/v0/archiveInsight [post]
func (h archiveInsightHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if r.ParseForm() != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, errors.New("Wrong Input"))
	}

	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		return httperror.NewHTTPStatusCodeError(http.StatusUnprocessableEntity, err)
	}

	h.e.ArchiveInsight(int64(id))

	return nil
}

func HttpInsights(auth *auth.Authenticator, mux *http.ServeMux, timezone *time.Location, f core.Fetcher, e *insights.Engine) {
	recommendationURLContainer := recommendation.GetDefaultURLContainer()

	mux.Handle("/api/v0/fetchInsights",
		httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithInterval(timezone)).
			WithEndpoint(fetchInsightsHandler{f: f, recommendationURLContainer: recommendationURLContainer}))

	mux.Handle("/api/v0/rateInsight",
		httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout)).
			WithEndpoint(rateInsightHandler{e: e}))

	mux.Handle("/api/v0/archiveInsight",
		httpmiddleware.WithDefaultStack(auth, httpmiddleware.RequestWithTimeout(httpmiddleware.DefaultTimeout)).
			WithEndpoint(archiveInsightHandler{e: e}))
}
