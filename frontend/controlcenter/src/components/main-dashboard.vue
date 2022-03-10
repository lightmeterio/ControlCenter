<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="row dashboard">
    <dashboard-supergraph :graphDateRange="graphDateRange" endpoint="sentMailsByMailbox" :title="sentMailsByMailboTitle"></dashboard-supergraph>
    <dashboard-supergraph :graphDateRange="graphDateRange" endpoint="bouncedMailsByMailbox" :title="bouncedMailsByMailboxTitle"></dashboard-supergraph>
    <dashboard-supergraph :graphDateRange="graphDateRange" endpoint="deferredMailsByMailbox" :title="deferredMailsByMailboxTitle"></dashboard-supergraph>
    <dashboard-supergraph :graphDateRange="graphDateRange" endpoint="expiredMailsByMailbox" :title="expiredMailsByMailboxTitle"></dashboard-supergraph>
    <dashboard-supergraph :graphDateRange="graphDateRange" endpoint="receivedMailsByMailbox" :title="receivedMailsByMailboxTitle"></dashboard-supergraph>
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
      sentMailsByMailboTitle: this.$gettext("Sent Mails By Sender"),
      bouncedMailsByMailboxTitle:  this.$gettext("Bounced Mails By Sender"),
      deferredMailsByMailboxTitle: this.$gettext("Deferred Mails By Sender"),
      expiredMailsByMailboxTitle: this.$gettext("Expired Mails By Sender"),
      receivedMailsByMailboxTitle: this.$gettext("Received Mails By Sender")
    };
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
  display: grid;
  justify-items: stretch;
  grid-template-columns: 2fr 2fr;
}
</style>
