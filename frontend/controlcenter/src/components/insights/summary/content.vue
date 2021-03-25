<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div>
    <div class="summary-header" v-translate render-html="true">
    This is the summary of the mail activity between <strong>%{summaryFrom}</strong> and <strong>%{summaryTo}</strong>
    </div>
    <table class="table import-summary-table">
      <thead class="thead">
        <th scope="col"><translate>Date and Time</translate></th>
        <th scope="col"><translate>Issue</translate></th>
      </thead>
      <tbody>
        <tr v-for="insight in insightsWithComponents(content.insights)" v-bind:key="insight.insight.id">
          <td>
            <span class="date" v-html="formatTableDate(insight.insight.time)"></span>
            <span class="time" v-html="formatTableTime(insight.insight.time)"></span>
          </td>
          <td>
            <component v-bind:is="insight.component" :insight="insight.insight"></component>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
import moment from "moment";
import tracking from "../../../mixin/global_shared.js";

import messageRBL from "./message-rbl";
import localRBL from "./local-rbl";
import highBounceRate from "./high-bounce-rate";
import mailInactivity from "./mail-inactivity";
import emptyDescription from "./empty";

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

function format(time) {
  return moment(time).format("DD MMM YYYY");
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
    formatTableDate(time) {
      return moment(time).format("DD MMM YYYY").replaceAll(' ','&#160;');
    },
    formatTableTime(time) {
      return moment(time).format("h:mmA");
    }
  },
  computed: {
    summaryFrom() {
      return format(this.content.interval.from);
    },
    summaryTo() {
      return format(this.content.interval.to);
    }
  }
}
</script>

<style scoped lang="less">
* {
  font-size: 15px;
}

.import-summary-table .thead th {
  background-color: #5f689a;
  color: #ffffff;
}

.import-summary-table tr td {
  background: #F9F9F9 0% 0% no-repeat padding-box;
  .time::before {
    content: " | ";
  }
  @media (max-width: 768px) {
    .time {
      display: none;
    }
  }
}

.summary-header strong {
  background-color: #e8e9f0;
  border-radius: 11px;
  padding-left: 0.5em;
  padding-right: 0.5em;
}

.summary-header {
  margin-bottom: 20px;
}

</style>
