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
        color: [],
        title: {
          text: "Sent mail by mailbox"
        },
        tooltip: {
          trigger: "axis",
          axisPointer: {
            type: "cross",
            label: {
              backgroundColor: "#6a7985"
            }
          }
        },
        legend: {
        },
        toolbox: {
          feature: {
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
            data: []
          }
        ],
        yAxis: [
          {
            type: "value"
          }
        ],
        series: []
      }
    };
  },
  mounted() {
    this.redrawChart();
  },
  methods: {
    redrawChart() {
      let self = this;

      fetchSentMailsByMailboxDataWithTimeInterval('2000-01-01', '4000-01-01').then(function(response) {
        let times = response.data.times;
        let values = response.data.values;

        self.option.series = [];

        function randomColor() {
            var o = Math.round, r = Math.random, s = 255;
            return 'rgba(' + o(r()*s) + ',' + o(r()*s) + ',' + o(r()*s) + ',' + r().toFixed(1) + ')';
        }

        let series = [];

        let category = 'Counters';

        for (const [mailbox, counters] of Object.entries(values)) {
          let serie = {
            name: mailbox,
            type: "line",
            stack: category,
            smooth: true,
            lineStyle: {
              width: 0
            },
            showSymbol: false,
            label: {
              show: true,
              position: "top"
            },
            areaStyle: {
              opacity: 0.8,
              color: randomColor()
            },
            emphasis: {
              focus: "series"
            },
            data: counters.slice(3, 10)
          };

          series.push(serie);
        }

        console.log(times)

        self.$refs.chart.setOption({
          series: series,
          xAxis: {
            data: times.slice(3,10)
          }
        })
      })
    }
  }
};
</script>

<style scoped>
.chart {
  height: 600px;
}
</style>
