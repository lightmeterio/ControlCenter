<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div>
    <div>{{header}}</div>
    <ul>
      <li v-for="insight in insightsWithComponents(content.insights)" v-bind:key="insight.id">
      <component v-bind:is="insight.component" :insight="insight.insight"></component>
      </li>
    </ul>
  </div>
</template>

<script>
import moment from "moment";
import tracking from "../../../mixin/global_shared.js";
import linkify from 'vue-linkify';
import Vue from "vue";

import messageRBL from "./message-rbl";
import localRBL from "./local-rbl";
import highBounceRate from "./high-bounce-rate";
import mailInactivity from "./mail-inactivity";
import emptyDescription from "./empty";

Vue.directive('linkified', linkify);

function componentForType(insight) {
  switch (insight.content_type) {
    case "message_rbl":
      return messageRBL;
    case "local_rbl_check":
      return localRBL;
    case "high_bounce_rate":
      return highBounceRate;
    case "mail_inactivity":
      return mailInactivity;
    default:
      return emptyDescription;
  }
}

export default {
  mixins: [tracking],
  props: {
    content: Object
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
    insightsWithComponents(insights) {
      return insights.map(function(insight){ return {"insight": insight, "component": componentForType(insight) } });
    },
  },
  computed: {
    header() {
      let translated = this.$gettext(`This is the summary of the mail activity between %{from} and %{to}:`)
      let format = time => moment(time).format("YYYY-MM-DD");
      let message = this.$gettextInterpolate(translated, {"from": format(this.content.interval.from), "to": format(this.content.interval.to)});
      return message;
    }
  }
}
</script>
 
