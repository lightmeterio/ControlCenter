<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div id="reports-page" class="d-flex flex-column min-vh-100">
    <mainheader></mainheader>
    <div class="container main-content">
      <div class="row">
        <div class="col-md-12">
          <h2>
            <translate>Latest 20 reports shared with the network</translate>
          </h2>

          <p v-if="reports.length == 0">
            No reports have been sent yet.
          </p>

          <div
            class="report"
            v-for="(report, reportIndex) in reports"
            :key="reportIndex"
          >
            <div>
              <translate>Report</translate> #{{ report.id }}
              –
              <span :title="report.dispatch_time">{{
                humanDuration(report.dispatch_time)
              }}</span>
              –
              <button
                class="btn btn-sm"
                v-b-modal.modal-explanation
                v-on:click="
                  reportKindExplanationTitle = report.kind;
                  reportKindExplanation = explanation(report.kind);
                "
              >
                <span>{{ report.kind }}</span>
              </button>
            </div>
            <vue-json-pretty :data="report.value"> </vue-json-pretty>
          </div>

          <b-modal
            ref="modal-explanation"
            id="modal-explanation"
            hide-footer
            :title="reportKindExplanationTitle"
            cancel-only
          >
            {{ reportKindExplanation }}
          </b-modal>
        </div>
      </div>
    </div>
    <mainfooter></mainfooter>
  </div>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import { getLatestReports } from "../lib/api.js";

import tracking from "../mixin/global_shared.js";
import auth from "../mixin/auth.js";

import VueJsonPretty from "vue-json-pretty";
import "vue-json-pretty/lib/styles.css";

export default {
  name: "reports",
  components: {
    VueJsonPretty
  },
  mixins: [tracking, auth],
  data() {
    return {
      reports: [],
      reportKindExplanation: "...",
      reportKindExplanationTitle: "..."
    };
  },
  computed: {},
  methods: {
    loadReports() {
      let vue = this;

      getLatestReports().then(function(response) {
        vue.reports = response.data;
      });
    },
    humanDuration(sinceTime) {
      let seconds = Math.floor((Date.now() - Date.parse(sinceTime)) / 1000);
      if (seconds < 60) {
        let message = this.$gettext("%{s}s ago");
        return this.$gettextInterpolate(message, { s: seconds });
      } else if (seconds < 60 * 60) {
        let message = this.$gettext("%{m}m ago");
        return this.$gettextInterpolate(message, {
          m: Math.floor(seconds / 60)
        });
      } else if (seconds < 60 * 60 * 24) {
        let message = this.$gettext("%{h}h ago");
        return this.$gettextInterpolate(message, {
          h: Math.floor(seconds / 60 / 60)
        });
      }
      let message = this.$gettext("%{d}d ago");
      return this.$gettextInterpolate(message, {
        d: Math.floor(seconds / 60 / 60 / 24)
      });
    },
    explanation(report_kind) {
      let explanations = {
        connection_stats:
          "IP addresses and timestamps of incoming SMTP connections that try to authenticate on ports 587 or 465. This data is obtained from the Postfix logs. It is used to identify and help prevent brute force attacks via the SMTP protocol based on automated threat sharing between Lightmeter users.",
        insights_count:
          "General statistics about recently generated Lightmeter insights. This data is used to improve the frequency of insights generation, e.g. avoiding too many notifications.",
        log_lines_count:
          "General non-identifiable information about which kinds of logs Postfix generates, and whether they are currently supported by Lightmeter ControlCenter. This is used to identify and prioritise support for unsupported log events.",
        mail_activity:
          "Statistics about the number of sent, bounced, deferred, and received messages. This helps identify Lightmeter performance issues and benchmark network health.",
        top_domains:
          "Domains that Postfix is managing locally. This is used to verify that reports are authentic."
      };

      return explanations[report_kind] ? explanations[report_kind] : "...";
    }
  },
  mounted() {
    this.loadReports();
  },
  destroyed() {}
};
</script>

<style lang="less">
.report {
  margin: auto;
  margin-top: 1rem;
  padding-top: 1rem;

  & + .report {
    border-top: 1px dotted #bdc3c7;
  }

  .vjs-tree {
    display: inline-block;
  }
}
</style>
