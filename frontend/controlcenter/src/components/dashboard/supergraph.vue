<template>
  <v-chart ref="chart" class="chart" :option="option" />
</template>

<script>
import VChart, { THEME_KEY } from "vue-echarts";

import "echarts";

import { fetchSentMailsByMailboxDataWithTimeInterval } from "@/lib/api";

export default {
  name: "SuperGraph",
  props: {
    graphDateRange: Object,
    endpoint: String,
    title: String,
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
    return {
      option: {
        animation: false,
        title: {
          text: this.title
        },
        tooltip: {
          trigger: "axis",
          //name: "tooltip-1",
          //formatter: function(params) {
          //  return params.toString();
          //}
          axisPointer: {
            type: "line",
            label: {
              backgroundColor: "#6a7985"
            }
          }
        },
        toolbox: {
          feature: {
            dataZoom: {
              yAxisIndex: "none"
            },
            magicType: { type: ["line", "bar", "stack", "tiled"] }
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
            boundaryGap: false
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
            //tooltip: {
            //  trigger: "axis",
            //  name: "tooltip-1",
            //  formatter: function(params) {
            //    return params.toString();
            //  }
            //},
            data: []
          }
        ]
      }
    };
  },
  mounted() {
    // TODO: also have a look at this example: https://jsfiddle.net/b6yr1u79/
    //this.$refs.chart.chart.on('highlight', 'series.name', function(params) {
    //  // TODO: ob highlight, somehow obtain the series under the cursor to show only relevant information about it!
    //  console.log(params);
    //});

    this.redrawChart(
      this.graphDateRange.startDate,
      this.graphDateRange.endDate
    );
  },
  methods: {
    setupStuff() {},
    redrawChart(from, to) {
      let self = this;

      fetchSentMailsByMailboxDataWithTimeInterval(this.endpoint, from, to, 6).then(function(
        response
      ) {
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
            data: counters,
            tooltip: {
              //name: "tooltip-" + mailbox,
              //formatter: function(params) {
              //  return params.name;
              //}
            }
          };

          series.push(s);
        }

        let newOptions = {
          series: series,
          xAxis: {
            data: times
          }
        };

        self.$refs.chart.setOption(newOptions);
      });
    }
  }
};
</script>

<style lang="less" scoped>
.chart {
  height: 400px;
}
</style>
