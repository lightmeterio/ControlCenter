<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <span>{{message()}}</span>
</template>

<script>

import tracking from "../../../mixin/global_shared.js";
import linkify from 'vue-linkify';
import Vue from "vue";

Vue.directive('linkified', linkify);

export default {
  mixins: [tracking],
  props: {
    insight: Object
  },
  computed: {
  },
  updated() {
  },
  mounted() {
  },
  data() {
    return {
    };
  },
  methods: {
    message() {
      return this.$gettextInterpolate(this.$gettext("Blocked by %{host}"), {"host": this.insight.content.host})
    }
  }
}
</script>
