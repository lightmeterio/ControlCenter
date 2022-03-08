<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="row dashboard">
    <dashboard-supergraph></dashboard-supergraph>
  </div>
</template>

<script>
import tracking from "../mixin/global_shared.js";

export default {
  name: "mainhdashboard",
  mixins: [tracking],
  props: {
    graphDateRange: Object
  },
  data() {
    return {};
  },
  computed: {},
  beforeDestroy() {},
  mounted() {},
  watch: {},
  methods: {}
};
</script>
<style lang="less" scoped>
.dashboard {
  border: 1px dotted black;
}
</style>
