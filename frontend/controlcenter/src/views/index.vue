<template>
  <div id="insights-page" class="d-flex flex-column min-vh-100">
    <mainheader></mainheader>
    <div class="container main-content">
      <div class="row">
        <div class="col-md-12">
          <h1 class="row-title">
            Control Center<!--{{translate "Control Center"}}-->
          </h1>
        </div>
      </div>

      <div class="row">
        <div class="col-md-12">
          <div class="panel panel-default greeting">
            <div class="row">
              <div class="col-md-3 align-center">
                <img
                  class="hero"
                  src="@/assets/greeting-observatory.svg"
                  alt="Observatory illustration"
                />
              </div>

              <div class="col-md-9 d-flex align-items-center">
                <div class="row">
                  <div class="container">
                    <h3>{{ greetingText }}</h3>
                    <p>
                      and welcome back<!-- {{translate "and welcome back"}}-->
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="row">
        <div class="col-md-4">
          <div id="delivery-attempts" class="card">
            <div class="card-header">
              Delivery attempts
              <!-- {{translate "Delivery attempts"}} -->
            </div>
            <div class="card-body">
              <div class="dashboard-gadget" id="deliveryStatus"></div>
            </div>
          </div>
        </div>
        <div class="col-md-8">
          <b-tabs id="basic-graphs-area" content-class="mt-3" justified>
            <b-tab title="Busiest domains" active>
              <div class="dashboard-gadget" id="topBusiestDomains"></div
            ></b-tab>
            <b-tab title="Bounced Domains">
              <div class="dashboard-gadget" id="topBouncedDomains"></div
            ></b-tab>
            <b-tab title="Deferred Domains">
              <div class="dashboard-gadget" id="topDeferredDomains"></div>
            </b-tab>
          </b-tabs>
        </div>
      </div>

      <div class="row container d-flex time-interval card-section-heading">
        <div class="col-lg-2 col-md-6 col-6 p-2">
          <h2>
            Insights
            <!--{{translate "Insights"}}-->
          </h2>
        </div>
        <div class="col-lg-3 col-md-6 col-6 p-2">
          <label class="col-md-2 col-form-label sr-only">
            Time interval:
            <!--{{translate "Time interval:"}}--></label
          >
          <DateRangePicker
            @update="updateInterval"
            :autoApply="autoApply"
            :opens="opens"
            :single-date-picker="singleDatePicker"
            :always-show-calendars="alwaysShowCalendars"
            :ranges="ranges"
            v-model="dateRange"
            :showCustomRangeCalendars="false"
          >
          </DateRangePicker>
        </div>

        <div class="col-lg-4 col-md-6 col-12 ml-auto p-2">
          <form id="insights-form">
            <div
              class="form-group d-flex justify-content-end align-items-center"
            >
              <label class="sr-only"
                >Filter
                <!-- {{translate "Filter"}}: --></label
              >
              <select
                id="insights-filter"
                class="form-control custom-select custom-select-sm"
                name="filter"
                form="insights-form"
                v-model="insightsFilter"
                style="width: 33%"
                v-on:change="onFetchInsights"
              >
                <!-- todo remove in style -->
                <option selected value="nofilter"
                  >All
                  <!-- {{translate "All"}} --></option
                >
                <!--    onclick="_paq.push(['trackEvent', 'InsightsFilterCategoryHomepage', 'click', this.value]);" -->
                <option value="category-local"
                  >Local
                  <!-- {{translate "Local"}}--></option
                >
                <!--    onclick="_paq.push(['trackEvent', 'InsightsFilterCategoryHomepage', 'click', this.value]);" -->
                <option value="category-news"
                  >News
                  <!-- {{translate "News"}} -->
                </option>
                <!--    onclick="_paq.push(['trackEvent', 'InsightsFilterCategoryHomepage', 'click', this.value]);" -->
              </select>
              <select
                id="insights-sort"
                class="form-control custom-select custom-select-sm"
                name="order"
                form="insights-form"
                v-model="insightsSort"
                style="width: 38%"
                v-on:change="onFetchInsights"
              >
                <!-- todo remove in style -->
                <option selected value="creationDesc"
                  >Newest
                  <!-- {{translate "Newest"}}--></option
                >
                <!-- onclick _paq.push(['trackEvent', 'InsightsFilterOrderHomepage', 'click', this.value]);-->

                <option value="creationAsc"
                  >Oldest
                  <!--{{translate "Oldest"}}--></option
                >

                <!-- onclick _paq.push(['trackEvent', 'InsightsFilterOrderHomepage', 'click', this.value]);-->
              </select>
            </div>
          </form>
        </div>
      </div>
      <!-- {{translate `Info`}} -->
      <insights class="row" :insights="insights"></insights>
    </div>
    <mainfooter></mainfooter>
  </div>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import moment from "moment";
import Plotly from "plotly.js-dist";
import {
  fetchInsights,
  fetchGraphDataAsJsonWithTimeInterval
} from "../lib/api.js";
import DateRangePicker from "../3rd/components/DateRangePicker.vue";

function defaultRange() {
  let today = new Date();
  today.setHours(0, 0, 0, 0);
  let yesterday = new Date();
  yesterday.setDate(today.getDate() - 1);
  yesterday.setHours(0, 0, 0, 0);
  let thisMonthStart = new Date(today.getFullYear(), today.getMonth(), 1);
  let thisMonthEnd = new Date(today.getFullYear(), today.getMonth() + 1, 0);
  return {
    Today: [today, today],
    Yesterday: [yesterday, yesterday],
    "This month": [thisMonthStart, thisMonthEnd],
    "This year": [
      new Date(today.getFullYear(), 0, 1),
      new Date(today.getFullYear(), 11, 31)
    ]
  };
}

export default {
  name: "insight",
  components: { DateRangePicker },
  data() {
    return {
      autoApply: true,
      alwaysShowCalendars: false,
      singleDatePicker: "range",
      dateRange: {
        startDate: "",
        endDate: ""
      },
      ranges: defaultRange(),
      opens: "left",
      insightsFilter: "nofilter",
      insightsSort: "creationDesc",
      insights: [],
      graphAreaResizeObserver: null
    };
  },
  computed: {
    greetingText() {
      let dateObj = new Date();
      let weekday = dateObj.toLocaleString("default", { weekday: "long" });
      // "{{ translate `Happy %s,` }}"
      return "Happy " + weekday + ",";
    }
  },
  methods: {
    // updateDashboard and updateInterval are placeholder functions
    updateDashboard: function() {},
    updateInterval: function() {},
    onFetchInsights: function() {
      let vue = this;
      let s = moment(vue.dateRange.startDate).format("YYYY-MM-DD");
      let e = moment(vue.dateRange.endDate).format("YYYY-MM-DD");

      fetchInsights(s, e, vue.insightsFilter, vue.insightsSort).then(function(
        response
      ) {
        vue.insights = response.data;
      });
    },
    drawDashboard: function() {
      let vue = this;

      const updateArray = function(dst, src) {
        dst.splice(0, Infinity, ...src);
      };

      var resizers = [];

      var updateDonutChart = function(graphName) {
        var chartData = [
          {
            values: [],
            marker: {
              colors: [
                "rgb(135, 197, 40)",
                "rgb(255, 92, 111)",
                "rgb(118, 17, 195)",
                "rgb(122, 130, 171)"
              ]
            },
            labels: [],
            type: "pie",
            hole: 0.3
          }
        ];
        var layout = {
          height: 220,
          margin: {
            t: 20,
            l: 20,
            r: 20,
            b: 20
          }
        };

        Plotly.newPlot(graphName, chartData, layout, { responsive: true });

        return function(start, end) {
          fetchGraphDataAsJsonWithTimeInterval(start, end, graphName).then(
            function(response) {
              let d =
                response.data != null ? response.data.map(v => v["value"]) : [];
              let l =
                response.data != null ? response.data.map(v => v["key"]) : [];
              updateArray(chartData[0].values, d);
              updateArray(chartData[0].labels, l);
              Plotly.redraw(graphName);
            }
          );
        };
      };

      var updateBarChart = function(graphName) {
        var chartData = [
          {
            x: [],
            y: [],
            type: "bar",
            marker: {
              // TODO: find a more elegant solution for this
              color: [
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)",
                "rgb(149, 205, 234)"
              ]
            }
          }
        ];
        var layout = {
          height: 220,
          xaxis: {
            automargin: true
          },
          yaxis: {
            automargin: true
          },
          margin: {
            t: 0,
            l: 30,
            r: 0,
            b: 50
          }
        };

        Plotly.newPlot(graphName, chartData, layout, { responsive: true }).then(
          function() {
            resizers.push(function(dimension) {
              layout.width = dimension.contentRect.width;
              Plotly.redraw(graphName);
            });
          }
        );

        return function(start, end) {
          fetchGraphDataAsJsonWithTimeInterval(start, end, graphName).then(
            function(response) {
              let x =
                response.data != null ? response.data.map(v => v["key"]) : [];
              let y =
                response.data != null ? response.data.map(v => v["value"]) : [];
              updateArray(chartData[0].x, x);
              updateArray(chartData[0].y, y);
              Plotly.redraw(graphName);
            }
          );
        };
      };

      const updateDeliveryStatus = updateDonutChart(
        "deliveryStatus",
        "Delivery Status"
      );
      const updateTopBusiestDomainsChart = updateBarChart(
        "topBusiestDomains",
        "Busiest Domains"
      );
      const updateTopDeferredDomainsChart = updateBarChart(
        "topDeferredDomains",
        "Most Deferred Domains"
      );
      const updateTopBouncedDomainsChart = updateBarChart(
        "topBouncedDomains",
        "Most Bounced Domains"
      );

      vue.updateDashboard = function(start, end) {
        updateDeliveryStatus(start, end);
        updateTopBusiestDomainsChart(start, end);
        updateTopDeferredDomainsChart(start, end);
        updateTopBouncedDomainsChart(start, end);
      };

      // Plotly has a bug that makes it unable to resize hidden graphs:
      // https://github.com/plotly/plotly.js/issues/2769
      // We try to workaround it
      var setupResizers = (function() {
        // Bail out, no support for ResizeObserver
        if (window.ResizeObserver === undefined) {
          return function() {};
        }

        let graphAreaResizeObserver = new ResizeObserver(function(entry) {
          for (let cb in resizers) {
            resizers[cb](entry[0]);
          }
        });
        vue.graphAreaResizeObserver = graphAreaResizeObserver;
        return function(e) {
          graphAreaResizeObserver.observe(e);
        };
      })();

      setupResizers(document.getElementById("basic-graphs-area"));

      vue.updateInterval = function(obj) {
        let s = moment(obj.startDate).format("YYYY-MM-DD");
        let e = moment(obj.endDate).format("YYYY-MM-DD");
        vue.updateDashboard(s, e);
        fetchInsights(s, e, vue.insightsFilter, vue.insightsSort).then(function(
          response
        ) {
          vue.insights = response.data;
        });
      };

      vue.updateIntervalOnStart = function(start, end) {
        vue.dateRange.startDate = start;
        vue.dateRange.endDate = end;
        vue.updateDashboard(start, end);
        fetchInsights(start, end, vue.insightsFilter, vue.insightsSort).then(
          function(response) {
            vue.insights = response.data;
          }
        );
      };

      let start = moment().subtract(29, "days");
      const end = moment();

      vue.updateIntervalOnStart(
        start.format("YYYY-MM-DD"),
        end.format("YYYY-MM-DD")
      );
    }
  },
  beforeDestroy() {
    this.graphAreaResizeObserver.disconnect();
  },
  mounted() {
    this.drawDashboard();
  },
  destroyed() {}
};
</script>

<style lang="less">
#insights-page .greeting h3 {
  font: 22px/32px Inter;
  font-weight: bold;
  margin: 0;
  text-align: left;
}

#insights-page .greeting p {
  text-align: left;
}

#insights-page .card-section-heading {
  background-color: #f9f9f9;
}

#insights-page .time-interval {
  margin: 0.6rem 0 0 0;
  border-radius: 10px;
}

#insights-page .card-section-heading h2 {
  font-size: 24px;
  font-weight: bold;
  margin: 0;
}

#insights-page .time-interval .form-group {
  margin: 0;
  padding: 0;
}

#insights-page #insights-form select {
  font-size: 12px;
  border-radius: 5px;
  margin-right: 0.2rem;
}

#insights-page .form-control.custom-select {
  margin: 0;
  background-color: #e6e7e7;
  color: #202324;
}

#insights-page .greeting {
  background: url(~@/assets/greeting-lensflare.svg) no-repeat right top,
    linear-gradient(104deg, #2a93d6 0%, #3dd9d6 100%) 0% 0% padding-box;
  color: white;
  padding: 0.5rem;
  border-radius: 7px;
  margin-bottom: 30px;
}

#insights-page h1.row-title {
  font-size: 32px;
  font-weight: bold;
  margin: 0.7em 0 0.8em 0;
  text-align: left;
}

#insights-page .vue-daterange-picker .reportrange-text {
  background: #daebf4;
  cursor: pointer;
  padding: 0.3rem 1rem;
  border: none;
  font-size: 12px;
  color: #00689d;
  font-weight: bold;
  border-radius: 5px;
  text-align: center;
  cursor: pointer;
}

#insights-page .vue-daterange-picker .reportrange-text {
  display: flex;
  justify-content: center;
}

#insights-page .vue-daterange-picker .reportrange-text span {
  order: 1;
  margin-top: 0.25em;
}

#insights-page .vue-daterange-picker .reportrange-text svg {
  order: 2;
  margin-left: 1em;
  margin-top: 0.45em;
}

#insights-page .modebar {
  display: none;
}

#insights-page #delivery-attempts .card-header {
  text-align: left;
  font-size: 15px;
  font-weight: bold;
  font-family: Inter;
  color: #202324;
}

#insights-page #delivery-attempts .card-header {
  background: none;
  border: none;
}

#insights-page #delivery-attempts {
  background: none;
  border: none;
}

#insights-page .tabs .nav-link.active {
  color: #fff;
  background: #1d8caf 0% 0% no-repeat padding-box;
  border: none;
  border-radius: 27px;
}

#insights-page .tabs .nav-tabs {
  border-bottom: none;
}

#insights-page .nav-tabs .nav-link {
  color: #1d8caf;
  font-size: 15px;
  font-weight: bold;
  font-family: Inter;
}

#insights-page .nav-tabs .nav-link:hover {
  border: 1px solid #95cdea;
}

#insights-page .tabs .nav-tabs a:hover {
  border: none;
}

@media (min-width: 768px) {
  #insights-page .greeting {
    height: 150px;
  }
}
</style>
