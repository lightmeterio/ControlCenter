<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div ref="chart" :class="chartClass()">
    <div
      ref="overlay"
      class="small-chart-overlay"
      @click="zoomIn()"
      v-on:keyup.enter="zoomOut()"
    ></div>
    <v-chart ref="echart" class="chart" :option="option" />
  </div>
</template>

<script>
import VChart, { THEME_KEY } from "vue-echarts";

import "echarts";
import moment from "moment";
import { fetchSentMailsByMailboxDataWithTimeInterval } from "@/lib/api";

export default {
  name: "SuperGraph",
  props: {
    graphDateRange: Object,
    endpoint: String,
    title: String,
    size: String
  },
  watch: {
    graphDateRange: {
      handler(graphDateRange) {
        this.redrawChart(graphDateRange.startDate, graphDateRange.endDate);
      },
      deep: true
    }
  },
  components: {
    VChart
  },
  provide: {
    [THEME_KEY]: "default"
  },
  data() {
    let vue = this;
    return {
      granularity: false,
      zoomed: false,
      option: {
        animation: false,
        title: {
          text: vue.title
        },
        tooltip: {
          trigger: "axis",
          position: "inside",
          triggerOn: "click",
          hideDelay: 2000,
          enterable: true,
          confine: true,
          formatter: function(params) {
            let dateDisplayed = false;
            let tt = "<div class='lm-tooltip'>";

            params.sort((s1, s2) => s1.value < s2.value);
            params.forEach(function(s) {
              if (s.value == 0) {
                return;
              }

              let date = vue.formatTime(s.axisValue, true);

              if (!dateDisplayed) {
                dateDisplayed = true;
                tt +=
                  "<p><b>" + date + " </b>" + vue.detectiveLink(date) + "</p>";
              }

              tt += "<div class='lm-serie'>";

              tt +=
                "<span class='lm-serieName' style='color: " +
                s.color +
                "'>" +
                s.seriesName +
                "</span>";
              tt +=
                "<div>" +
                "<span class='lm-serieValue'>" +
                s.value +
                " </span>" +
                vue.detectiveLink(date, s.seriesName) +
                "</div>";
              tt += "</div>";
            });
            tt += "</div>";
            return tt;
          }
        },
        toolbox: {
          feature: {
            dataZoom: {
              yAxisIndex: "none"
            },
            magicType: { type: ["line", "bar"] }
          }
        },
        grid: {
          left: "3%",
          right: "4%",
          bottom: "3%",
          containLabel: true
        },
        xAxis: [
          {
            type: "category",
            boundaryGap: false,
            axisLabel: {
              formatter: this.formatTime
            }
          }
        ],
        yAxis: [{ type: "value" }],
        series: [
          // this is a dummy dataset to force the graph to stack the data
          {
            type: "line",
            stack: "Total",
            showSymbol: false,
            areaStyle: {
              opacity: 0.8,
              color: "black"
            },
            data: []
          }
        ]
      }
    };
  },
  mounted() {
    // TODO: also have a look at this example: https://jsfiddle.net/b6yr1u79/
    //this.$refs.echart.chart.on('highlight', 'series.name', function(params) {
    //  // TODO: ob highlight, somehow obtain the series under the cursor to show only relevant information about it!
    //  console.log(params);
    //});

    let vue = this;
    window.addEventListener("resize", function() {
      vue.$refs.echart.resize();
    });

    window.addEventListener("keydown", this.keyListener);

    this.redrawChart(
      this.graphDateRange.startDate,
      this.graphDateRange.endDate
    );
  },
  methods: {
    chartClass() {
      let bootstrapClass = {
        2: "col-md-6 col-12",
        3: "col-md-4 col-12"
      }[this.size];
      return "small-chart " + bootstrapClass + (this.zoomed ? " zoomed" : "");
    },
    zoomIn() {
      this.zoomed = true;
      this.$refs.chart.style.top = "" + window.scrollY + "px";
      setTimeout(this.$refs.echart.resize, 50);
    },
    keyListener(event) {
      if (event.key === "Escape") {
        this.zoomOut();
      }
    },
    zoomOut() {
      this.zoomed = false;
      this.$refs.chart.style.top = 0;
      setTimeout(this.$refs.echart.resize, 50);
    },
    formatTime(value, keepDate = false) {
      let val = new Date(value);

      if (this.granularity == 24) {
        return val.toISOString().substring(0, 10);
      }

      return keepDate === true
        ? val.toISOString().substring(0, 16)
        : val.toISOString().substring(11, 16);
    },
    detectiveLink(date, seriesName) {
      let vue = this;

      let status = {
        sentMailsByMailbox: 0,
        receivedMailsByMailbox: 42,
        bouncedMailsByMailbox: 1,
        deferredMailsByMailbox: 2,
        expiredMailsByMailbox: 3
      }[vue.endpoint];

      let from_to = vue.endpoint == "receivedMailsByMailbox" ? "to" : "from";

      let link =
        window.location.pathname +
        "#/detective?" +
        (seriesName
          ? "mail_" + from_to + "=" + encodeURIComponent(seriesName)
          : "") +
        "&startDate=" +
        encodeURIComponent(date) +
        "&endDate=" +
        encodeURIComponent(date) +
        "&statusSelected=" +
        encodeURIComponent(status);

      return (
        "<a href='" +
        link +
        "'> <i class='fas fa-search' data-toggle='tooltip' data-placement='bottom'></i></a>"
      );
    },
    redrawChart(from, to) {
      let vue = this;

      vue.granularity = (function() {
        // same day, no second precision
        if (from == to) {
          return 1;
        }

        let fromTime = moment(from);
        let toTime = moment(to);
        let diff = toTime.diff(fromTime, "hours");

        // one day or less, use hourly data
        if (diff <= 24) {
          return 1;
        }

        // over one day, use daily data
        return 24;
      })();

      fetchSentMailsByMailboxDataWithTimeInterval(
        this.endpoint,
        from,
        to,
        vue.granularity
      ).then(function(response) {
        let times = response.data.times.map(ts => new Date(ts * 1000));
        let values = response.data.values;

        let series = [];

        for (const [mailbox, counters] of Object.entries(values)) {
          let s = {
            name: mailbox,
            type: "line",
            stack: "Total",
            lineStyle: {
              width: 1
            },
            showSymbol: false,
            label: {
              show: true,
              position: "top"
            },
            areaStyle: {},
            emphasis: {
              focus: "series"
            },
            data: counters
          };

          series.push(s);
        }

        let newOptions = {
          series: series,
          xAxis: {
            data: times
          }
        };

        vue.$refs.echart.setOption(newOptions);
      });
    }
  }
};
</script>

<style lang="less" scoped>
.small-chart:not(.zoomed) {
  height: 300px;
  max-height: 66vh;
  padding: 1em;
  position: relative;

  .small-chart-overlay {
    width: 100%;
    height: 100%;
    position: absolute;
    z-index: 10;
    cursor: zoom-in;
    &:hover {
      background-color: #0069d9;
      opacity: 0.05;
    }
  }

  .chart {
    width: 100%;
    height: 100%;
  }
}
.small-chart.zoomed {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  max-width: 100%; /* override bootstrap col- */
  height: 100%;
  padding: 2em;
  z-index: 100;
  background: white;
}
</style>

<!-- NOTE: following CSS is not scoped on purpose: the echarts tooltip is just below <body> -->
<style lang="less">
.lm-serie {
  font-weight: bold;
  display: flex;
  justify-content: space-between;
  .lm-serieValue {
    margin-left: 0.5em;
  }
}
</style>
