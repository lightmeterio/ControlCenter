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
    >
      <translate v-if="emptyData">Not enough data to create graph!</translate>
    </div>
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
      emptyData: false,
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
            magicType: { type: ["line", "bar"] },
            // NOTE: it must start with "my", as described on
            // https://echarts.apache.org/en/option.html#toolbox.feature
            myExitButton: {
              show: true,
              title: "Close",
              // TODO: this is the output of a SVG path. The way I did was to draw it on inkscape and then copying from the exported SVG... Ugly :-(
              icon:
                "path://m 56.263973,119.44565 c -3.197323,-2.26186 -5.647055,-5.50354 -8.76242,-7.84006 -1.773664,-1.33025 -3.818239,-2.2855 -5.534163,-3.68944 -5.229431,-4.27863 -9.000974,-10.623068 -11.990683,-16.602487 -0.323485,-0.646972 0,-2.508668 0,-3.228261 0,-1.668478 -0.403702,-4.380529 0,-5.995341 0.3377,-1.350796 3.715498,-6.935074 5.072983,-7.840062 0.41801,-0.278673 1.054089,-0.263459 1.38354,-0.922359 0.06875,-0.137499 -0.108701,-0.35248 0,-0.461182 2.178109,-2.178108 6.729579,-2.767081 9.684782,-2.767081 1.08743,0 2.660772,-0.293905 3.68944,0 3.079483,0.879853 6.791521,3.828653 8.762423,6.456524 0.201126,0.268168 1.662917,2.403469 1.84472,2.767078 1.173512,2.347026 -2.286124,-0.902583 0.922361,2.305902 0.108702,0.108702 0.352478,-0.108701 0.46118,0 2.36465,2.36465 3.573793,6.109579 4.611801,9.223603 0.131453,0.394358 0.851336,1.489596 0.922361,1.84472 0.582941,2.91471 0.0099,3.24799 1.383541,5.995341 0.88415,1.768305 -0.736246,1.292538 1.84472,3.228265 2.400083,1.80006 0.02215,1.39092 2.767081,2.3059 0.672637,0.22421 1.582441,0 2.305899,0 0.854014,0 2.501085,0.36359 3.228261,0 3.559516,-1.77976 6.871221,-11.944204 10.145964,-15.218946 0.783854,-0.783858 1.901457,-1.152224 2.767078,-1.844723 1.765835,-1.412666 -1.314563,0.853387 0.922362,-1.383538 0.788418,-0.788419 2.267727,-2.080985 3.22826,-2.767081 7.731536,-5.522526 13.554556,-5.387335 18.447206,2.767081 1.28535,2.142252 1.38354,5.31391 1.38354,7.840062 0,4.243768 0.0868,7.416905 -2.3059,10.607145 -1.31494,1.75326 -2.6641,3.44318 -4.6118,4.6118 -2.52894,1.51736 -9.998295,4.1597 -11.529505,6.45652 -0.394142,0.59121 -0.219517,3.49207 0,4.15062 0.379677,1.13903 1.524532,1.85549 2.305902,2.76708 2.116583,2.46934 4.775793,4.46981 6.917703,6.9177 1.0123,1.15692 1.68007,2.60243 2.76708,3.68944 0.24306,0.24307 0.6793,0.21812 0.92236,0.46118 1.22849,1.2285 1.58958,3.54219 1.84472,5.07299 0.9509,5.70539 3.54971,10.59514 -1.84472,15.21894 -1.48968,1.27687 -3.63272,3.14617 -5.53416,3.68944 -3.0494,0.87126 -7.12851,0.86213 -10.145966,0 -1.084102,-0.30974 -2.136664,-1.11064 -3.228261,-1.38354 -0.298273,-0.0746 -0.636897,0.11419 -0.922359,0 -0.731607,-0.29264 -1.590709,-1.14514 -2.305902,-1.38354 -0.291676,-0.0972 -0.65872,0.15818 -0.922361,0 -0.684919,-0.41095 -1.643809,-1.64381 -2.305899,-2.3059 -0.614908,-0.61491 -1.45582,-1.06692 -1.84472,-1.84472 -0.89925,-1.7985 -1.904979,-3.34878 -2.767081,-5.07298 -0.309288,-0.61858 -0.170466,-1.72448 -0.46118,-2.3059 -0.19445,-0.38891 -0.727911,-0.53346 -0.922361,-0.92236 -0.283459,-0.56692 -0.177721,-1.27781 -0.461179,-1.84472 -0.355211,-0.71043 -1.806184,-2.26737 -2.305902,-2.76708 -0.278503,-0.27851 -2.329731,0 -2.767079,0 -0.515001,0 -1.360202,-0.16151 -1.844723,0 -2.579922,0.85997 -5.922065,7.03837 -7.378879,9.2236 -0.269656,0.40448 -0.169506,0.99464 -0.461182,1.38354 -0.521767,0.69569 -1.362348,1.12116 -1.84472,1.84472 -2.03426,3.05138 -4.451297,6.87757 -8.301241,7.84006 -1.193097,0.29828 -2.471989,-0.17392 -3.689443,0 -0.340289,0.0486 -0.578617,0.46118 -0.922359,0.46118 -1.539533,0 -4.75062,-0.45 -5.995341,-1.38354 -1.759874,-1.3199 -4.660932,-4.85746 -5.072981,-6.9177 -0.463357,-2.31678 -1.16786,-14.94363 -0.461182,-17.06367 0.06875,-0.20624 0.307454,-0.30745 0.461182,-0.46118 0.153726,-0.46118 0.343276,-0.91192 0.461179,-1.38354 0.45956,-1.83823 0.348242,-4.49886 1.84472,-5.99534 1.259438,-1.25944 5.917758,-3.97103 7.840062,-4.6118 0.547592,-0.18253 1.276162,0.11371 1.84472,0 2.172936,-0.43459 4.306477,-0.38485 6.456521,-0.92236 z",
              onclick: function() {
                vue.zoomOut();
              }
            }
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
      vue.resizeAndRefresh();
    });

    window.addEventListener("keydown", this.keyListener);

    this.redrawChart(
      this.graphDateRange.startDate,
      this.graphDateRange.endDate
    );
  },
  methods: {
    resizeAndRefresh() {
      this.$refs.echart.resize();
      //this.$refs.echart.refresh();
    },
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
      setTimeout(this.resizeAndRefresh, 50);
    },
    keyListener(event) {
      if (event.key === "Escape") {
        this.zoomOut();
      }
    },
    zoomOut() {
      this.zoomed = false;
      this.$refs.chart.style.top = 0;
      setTimeout(this.resizeAndRefresh, 50);
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

      // TODO: the graph should not know about each concrete usage of it.
      // Right now it's quite difficult, so a way to model it is not to include the link
      // to the detective page, but to emit a generic event with the time range, sender/recipient
      // and catch the event in the dashboard, which will act accordingly.
      // I am not sure though if this can be done with the tooltip formatter!
      let status = {
        sentMailsByMailbox: 0,
        receivedMailsByMailbox: 42,
        inboundRepliesByMailbox: 43,
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

        vue.emptyData = series.length == 0;

        // FIXME: Ugly hack due a bug on echarts: https://github.com/apache/echarts/issues/6202
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

    display: flex;
    justify-content: center;
    align-items: center;
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
