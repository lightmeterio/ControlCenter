<template>
  <v-chart ref="chart" class="chart" :option="option" />
</template>

<script>
import VChart, { THEME_KEY } from "vue-echarts";

import "echarts";

import { fetchSentMailsByMailboxDataWithTimeInterval } from "@/lib/api";

//function randomColor() {
//  var o = Math.round,
//    r = Math.random,
//    s = 255;
//  return (
//    "rgba(" +
//    o(r() * s) +
//    "," +
//    o(r() * s) +
//    "," +
//    o(r() * s) +
//    "," +
//    r().toFixed(1) +
//    ")"
//  );
//}

export default {
  name: "SuperGraph",
  props: {
    graphDateRange: Object
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
          text: "Sent mails per mailbox"
        },
        tooltip: {
          trigger: "axis",
          axisPointer: {
            type: "line",
            label: {
              backgroundColor: "#6a7985"
            }
          }
          //formatter: function(params) {
          //  console.log(params);
          //  return params[0].seriesName;
          //}
        },
        toolbox: {
          feature: {
            dataZoom: {
              yAxisIndex: 'none'
            },
            magicType: { type: ['line', 'bar', 'stack', 'tiled'] },
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
        yAxis: [
          {
            type: "value"
          }
        ],
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
            emphasis: {
              focus: "series"
            },
            data: []
          }
        ]
      }
    };
  },
  mounted() {
    this.redrawChart(this.graphDateRange.startDate, this.graphDateRange.endDate);
  },
  methods: {
    redrawChart(from, to) {
      let self = this;

      console.log("cacatua", from, to);

      fetchSentMailsByMailboxDataWithTimeInterval(from, to, 6).then(function(response) {
        let times = response.data.times.map(ts => new Date(ts * 1000));
        let values = response.data.values;

        let series = [];

        for (const [mailbox, counters] of Object.entries(values)) {
          let serie = {
            name: mailbox,
            type: "line",
            stack: "Total",
            lineStyle: {
              width: 0
            },
            showSymbol: false,
            label: {
              show: true,
              position: "top"
            },
            areaStyle: {
              //color: randomColor()
            },
            emphasis: {
              focus: "series"
            },
            data: counters
          };

          series.push(serie);
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

<style scoped>
.chart {
  height: 600px;
}
</style>
