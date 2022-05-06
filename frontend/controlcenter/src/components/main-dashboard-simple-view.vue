<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="row dashboard">
    <dashboard-supergraph
      :graphDateRange="graphDateRange"
      endpoint="sentMailsByMailbox"
      :title="sentMailsByMailboTitle"
      size="2"
    ></dashboard-supergraph>
    <dashboard-supergraph
      :graphDateRange="graphDateRange"
      endpoint="inboundRepliesByMailbox"
      :title="inboundRepliesByMailboxTitle"
      size="2"
    ></dashboard-supergraph>
    <dashboard-supergraph
      :graphDateRange="graphDateRange"
      endpoint="bouncedMailsByMailbox"
      :title="bouncedMailsByMailboxTitle"
      size="2"
    ></dashboard-supergraph>
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
    return {
      sentMailsByMailboTitle: this.$gettext("Sent"),
      bouncedMailsByMailboxTitle: this.$gettext("Bounced"),
      expiredMailsByMailboxTitle: this.$gettext("Expired"),
      inboundRepliesByMailboxTitle: this.$gettext("Replies")
    };
  }
};
</script>

<style lang="less" scoped>
.dashboard {
  margin: 2em 0;
}
</style>
