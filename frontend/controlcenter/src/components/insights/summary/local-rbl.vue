<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <span>{{ message }}</span>
</template>

<script>
import tracking from "../../../mixin/global_shared.js";

export default {
  mixins: [tracking],
  props: {
    insight: Object
  },
  computed: {
    message() {
      let message = this.$gettext(`Your IP address is listed by %{count} RBLs`);
      return this.$gettextInterpolate(message, {
        count: this.insight.content.rbls.length
      });
    }
  }
};
</script>
