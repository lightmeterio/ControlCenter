<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <span>{{ message }}</span>
</template>

<script>
import moment from "moment";
import tracking from "../../../mixin/global_shared.js";

export default {
  mixins: [tracking],
  props: {
    insight: Object
  },
  computed: {
    message() {
      let translated = this.$gettext(
        `Bounce rate of %{rate}% between %{from} and %{to}`
      );

      let format = time => moment(time).format("DD MMM | hh:mmA");

      return this.$gettextInterpolate(translated, {
        rate: this.insight.content.value * 100,
        from: format(this.insight.content.interval.from),
        to: format(this.insight.content.interval.to)
      });
    }
  }
};
</script>
