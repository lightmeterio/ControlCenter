<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div>
    <div v-html="header" class="blockedips-header"></div>
    <table class="table blockedips-summary-table">
      <thead class="thead">
        <th scope="col"><translate>Day</translate></th>
        <th scope="col"><translate># IPs</translate></th>
        <th scope="col"><translate># Connections</translate></th>
      </thead>
      <tbody>
        <tr v-for="rec in content.summary" v-bind:key="rec.time_interval">
          <td>
            {{ formatTableDay(rec.time_interval) }}
          </td>
          <td>
            {{ new Intl.NumberFormat().format(rec.ip_count) }}
          </td>
          <td>
            {{ new Intl.NumberFormat().format(rec.connections_count) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script>
import moment from "moment";
import tracking from "../../mixin/global_shared.js";

export default {
  mixins: [tracking],
  props: {
    content: Object
  },
  updated() {},
  mounted() {},
  data() {
    return {};
  },
  methods: {
    formatTableDate(time) {
      return moment(time).format("DD MMM YYYY");
    },
    formatTableDay(interval) {
      return moment(interval.from).format("DD MMM");
    }
  },
  computed: {
    header() {
      let translation = this.$gettext(
        `Summary of banned IPs from <strong>%{from}</strong> to <strong>%{to}</strong> due to potential attacks (brute-force, slow-force, botnets):`
      );
      return this.$gettextInterpolate(translation, {
        from: this.formatTableDate(this.content.time_interval.from),
        to: this.formatTableDate(this.content.time_interval.to)
      });
    }
  }
};
</script>

<style scoped lang="less">
* {
  font-size: 15px;
}

.blockedips-summary-table .thead th {
  background-color: #5f689a;
  color: #ffffff;
}

.blockedips-summary-table tr td {
  background: #f9f9f9 0% 0% no-repeat padding-box;
  .time::before {
    content: " | ";
  }
  @media (max-width: 768px) {
    .time {
      display: none;
    }
  }
}

.blockedips-header strong {
  background-color: #e8e9f0;
  border-radius: 11px;
  padding-left: 0.5em;
  padding-right: 0.5em;
}

.blockedips-header {
  margin-bottom: 20px;
}
</style>
