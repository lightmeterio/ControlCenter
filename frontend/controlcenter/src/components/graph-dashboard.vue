<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<template>
  <div id="graph-dashboard" class="row">
    <div class="col-md-4">
      <div id="delivery-attempts" class="card">
        <div class="card-header">
          <!-- prettier-ignore -->
          <translate>Delivery attempts</translate>
        </div>
        <div class="card-body">
          <div class="dashboard-gadget" id="deliveryStatus"></div>
        </div>
      </div>
    </div>
    <div class="col-md-8">
      <b-tabs id="basic-graphs-area" content-class="mt-3" justified>
        <b-tab
          v-on:click="trackEvent('change-domains-tab', 'topBusiestDomains')"
          :title="BusiestDomainsTitle"
          active
        >
          <div class="dashboard-gadget" id="topBusiestDomains"></div
        ></b-tab>
        <b-tab
          v-on:click="trackEvent('change-domains-tab', 'topBouncedDomains')"
          :title="BouncedDomainsTitle"
        >
          <div class="dashboard-gadget" id="topBouncedDomains"></div
        ></b-tab>
        <b-tab
          v-on:click="trackEvent('change-domains-tab', 'topDeferredDomains')"
          :title="DeferredDomainsTitle"
        >
          <div class="dashboard-gadget" id="topDeferredDomains"></div>
        </b-tab>
      </b-tabs>
    </div>
  </div>
</template>

<script>
import Plotly from "plotly.js-dist";
import { fetchGraphDataAsJsonWithTimeInterval } from "@/lib/api";
import tracking from "../mixin/global_shared.js";

export default {
  name: "graphdashboard",
  mixins: [tracking],
  props: {
    graphDateRange: Object
  },
  data() {
    return {
      graphAreaResizeObserver: null
    };
  },
  computed: {
    BusiestDomainsTitle: function() {
      return this.$gettext("Busiest Domains");
    },
    BouncedDomainsTitle: function() {
      return this.$gettext("Bounced Domains");
    },
    DeferredDomainsTitle: function() {
      return this.$gettext("Deferred Domains");
    }
  },
  beforeDestroy() {
    this.graphAreaResizeObserver.disconnect();
  },
  mounted() {
    this.drawDashboard();
  },
  watch: {
    graphDateRange: {
      handler(graphDateRange) {
        this.updateDashboard(graphDateRange.startDate, graphDateRange.endDate);
      },
      deep: true
    }
  },
  methods: {
    updateDashboard: function() {},
    drawDashboard: function() {
      let vue = this;

      const updateArray = function(dst, src) {
        dst.splice(0, Infinity, ...src);
      };

      let resizers = [];

      let updateDonutChart = function(graphName) {
        let chartData = [
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
        let layout = {
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

      let updateBarChart = function(graphName) {
        let chartData = [
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
        let layout = {
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

      const updateDeliveryStatus = updateDonutChart("deliveryStatus");
      const updateTopBusiestDomainsChart = updateBarChart("topBusiestDomains");
      const updateTopDeferredDomainsChart = updateBarChart("topDeferredDomains");
      const updateTopBouncedDomainsChart = updateBarChart("topBouncedDomains");

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

      vue.updateDashboard(
        vue.graphDateRange.startDate,
        vue.graphDateRange.endDate
      );
    }
  }
};
</script>
<style>
#graph-dashboard #delivery-attempts .card-header {
  text-align: left;
  font-size: 15px;
  font-weight: bold;
  font-family: Inter;
  color: #202324;
}

#graph-dashboard #delivery-attempts .card-header {
  background: none;
  border: none;
}

#graph-dashboard #delivery-attempts {
  background: none;
  border: none;
}

#graph-dashboard .tabs .nav-link.active {
  color: #fff;
  background: #1d8caf 0% 0% no-repeat padding-box;
  border: none;
  border-radius: 27px;
}

#graph-dashboard .tabs .nav-tabs {
  border-bottom: none;
}

#graph-dashboard .nav-tabs .nav-link {
  color: #1d8caf;
  font-size: 15px;
  font-weight: bold;
  font-family: Inter;
}

#graph-dashboard .nav-tabs .nav-item a:hover {
  border: 1px solid #95cdea;
  border-radius: 27px;
}
</style>
