<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<!-- More documentation at https://github.com/setaman/vue-ellipse-progress -->

<template>
  <vue-ellipse-progress
    :progress="value"
    :loading="!active && value < 100"
    color="#000500"
    emptyColor="#777777"
    :size="190"
    :thickness="20"
    lineMode="out 6"
    animation="rs 70 1000"
    fontSize="1.7rem"
    fontColor="red"
    >
  </vue-ellipse-progress>
</template>

<script>

import tracking from "../mixin/global_shared.js";
import { getAPI } from "@/lib/api";

export default {
  name: "import-progress-indicator",
  mixins: [tracking],
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
  mounted() {
    let vue = this;

    this.updateValue = window.setInterval(function() {
      getAPI("importProgress").then(function(progress) {
        let data = progress.data;
        let finished = (vue.active || (!vue.active && vue.value == 100)) && !data.active;

        vue.time = data.time;
        vue.value = data.value;
        vue.active = data.active;

        if (finished) {
          window.setTimeout(function() {
            window.clearInterval(vue.updateValue);
            vue.$emit("finished", vue)
          }, 400)
        }
      }).catch(function() {
        console.log("Error!!! obtaining progress");
      })
    }, 1000);
  },
  destroyed() {
    console.log("Oh My! I am dying now!");
    window.clearInterval(this.updateValue);
  }
};
</script>

<style lang="less">
</style>
