<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <span>{{message()}}</span>
</template>

<script>

import moment from "moment";
import tracking from "../../../mixin/global_shared.js";
import linkify from 'vue-linkify';
import Vue from "vue";

Vue.directive('linkified', linkify);

export default {
  mixins: [tracking],
  props: {
    insight: Object
  },
  methods: {
    message() {
      let translated = this.$gettext(`Bounce rate of %{rate}% between %{from} and %{to}`)

      let format = time => moment(time).format("YYYY-MM-DD hh:mm")

      return this.$gettextInterpolate(translated, {
        rate: this.insight.content.value * 100,
        from: format(this.insight.content.interval.from),
        to: format(this.insight.content.interval.to)
      })
    }
  }
}
</script>
 
