package api

import (
	"gitlab.com/lightmeter/controlcenter/insights/core"
	"gitlab.com/lightmeter/controlcenter/util"
	"net/http"
	"strconv"
	"time"
)

type fetchInsightsHandler struct {
	f        core.Fetcher
	timezone *time.Location
}

// @Summary Fetch Insights
// @Produce json
// @Param from query string true "Initial date in the format 1999-12-23"
// @Param to   query string true "Final date in the format 1999-12-23"
// @Param filter query string false "Filter by. Possible values: 'category'" Enums{"category"}
// @Param order query string true "Order by. Possible values: 'creationAsc', 'creationDesc'" Enums{"creationAsc", "creationDesc"}
// @Param entries query int false "Maximum number of insights to fetch"
// @Param category query string false "If filter by category, the category name. Possible values: 'info', 'warning', 'urgent'" Enums{"info", "warning", "urgent"}
// @Success 200 {object} fetchedInsight
// @Failure 422 {string} string "desc"
// @Router /api/v0/fetchInsights [get]
func (h fetchInsightsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ParseForm() != nil {
		http.Error(w, "Wrong input", http.StatusUnprocessableEntity)
		return
	}

	interval, err := intervalFromForm(h.timezone, r.Form)

	if err != nil {
		http.Error(w, "Error parsing time interval:\""+err.Error()+"\"", http.StatusUnprocessableEntity)
		return
	}

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
		http.Error(w, "Error parsing time interval:\""+err.Error()+"\"", http.StatusUnprocessableEntity)
		return
	}

	fetchedInsights, err := h.f.FetchInsights(core.FetchOptions{
		Interval:   interval,
		Category:   category,
		FilterBy:   filter,
		OrderBy:    order,
		MaxEntries: entries,
	})

	util.MustSucceed(err, "error fetching insights")

	insights := fetchInsightsResult{}

	for _, fi := range fetchedInsights {
		i := fetchedInsight{
			ID:          fi.ID(),
			Time:        fi.Time(),
			Priority:    int(fi.Priority()),
			Category:    fi.Category().String(),
			ContentType: fi.ContentType(),
			Content:     fi.Content(),
		}

		insights = append(insights, i)
	}

	serveJson(w, r, insights)
}

type fetchedInsight struct {
	ID          int
	Time        time.Time
	Priority    int
	Category    string
	ContentType string
	Content     interface{}
}

type fetchInsightsResult []fetchedInsight

func HttpInsights(mux *http.ServeMux, timezone *time.Location, f core.Fetcher) {
	mux.Handle("/api/v0/fetchInsights", fetchInsightsHandler{f: f, timezone: timezone})
}
