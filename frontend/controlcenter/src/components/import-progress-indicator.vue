<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<!-- More documentation at https://github.com/setaman/vue-ellipse-progress -->

<template>
  <div class="progress-indicator-area" v-show="shouldShowProgressIndicator">
    <div class="progress-indicator">
      <div class="ellipse">
        <vue-ellipse-progress
          line="square"
          :progress="value"
          emptyColor="#f9f9f9"
          empty-thickness="10"
          lineMode="normal"
          :loading="!active && value < 100"
          color="#2c9cd6"
          :size="150"
          :thickness="15"
          animation="rs 70 1000"
          fontSize="1.7rem"
          fontColor="black"
          :legend-value="value"
          :legend="true"
        >
          <span slot="legend-value">%</span>
        </vue-ellipse-progress>
      </div>
      <div class="generating-label text-center" v-show="showLabel">
        {{ label }}
      </div>
    </div>
  </div>
</template>

<script>
import tracking from "../mixin/global_shared.js";
import { getAPI } from "@/lib/api";
import { mapState } from "vuex";

export default {
  mixins: [tracking],
  props: {
    label: String,
    showLabel: {
      type: Boolean,
      default: true
    }
  },
  data() {
    return {
      time: "",
      value: 0,
      active: false,
      options: {
        color: "#000500",
        "empty-color": "#777777",
        size: 190,
        thickness: 20,
        "line-mode": "out 6",
        animation: "rs 70 1000",
        "font-size": "1.7rem",
        "font-color": "red"
      }
    };
  },
  computed: {
    shouldShowProgressIndicator: function() {
      return !this.isImportProgressFinished;
    },
    ...mapState(["isImportProgressFinished"])
  },
  mounted() {
    let vue = this;

    this.updateValue = window.setInterval(function() {
      getAPI("importProgress")
        .then(function(progress) {
          let data = progress.data;
          let finished =
            (vue.active || (!vue.active && vue.value == 100)) && !data.active;

          vue.time = data.time;
          vue.value = data.value;
          vue.active = data.active;

          if (!finished) {
            return;
          }

          window.setTimeout(function() {
            window.clearInterval(vue.updateValue);
            vue.$emit("finished", vue);
          }, 400);
        })
        .catch(function(error) {
          console.log("Error!!! obtaining progress");
          if (error.response.status === 401) {
            // this is when the user is not authenticated, e.g. an end-user using the Message Detective
            window.clearInterval(vue.updateValue);
            vue.$emit("finished", vue);
          }
        });
    }, 1000);
  },
  destroyed() {
    window.clearInterval(this.updateValue);
  }
};
</script>

<style scoped lang="less">
.generating-label {
  margin-top: 20px;
}

.progress-indicator .ellipse {
  margin: auto;
  display: flex;
}

.progress-indicator .ellipse > div {
  margin: 0 auto;
}
</style>
