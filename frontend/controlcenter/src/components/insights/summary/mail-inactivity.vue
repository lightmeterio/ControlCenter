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
      let format = time => moment(time).format(`DD MMM | hh:mmA`);
      let message = this.$gettext(`Mail inactivity from %{from} to %{to}`);

      return this.$gettextInterpolate(message, {
        from: format(this.insight.content.interval.from),
        to: format(this.insight.content.interval.to)
      });
    }
  }
};
</script>
