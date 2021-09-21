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

          <div class="metadata">
            <p>The following metadata is sent along with reports</p>
            <div class="metadata-table">
              <div>
                <span>
                  Instance ID
                  <i class="fas fa-info-circle" :title="InstanceIdText"></i>
                </span>
                <span>{{ instanceID }}</span>
              </div>
              <div>
                <span>Postfix IP</span>
                <span>{{ localIP }}</span>
              </div>
              <div>
                <span>Public URL</span>
                <span>{{ postfixURL }}</span>
              </div>
              <div>
                <span>Postfix Version</span>
                <span>{{ postfixVersion }}</span>
              </div>
              <div>
                <span>ControlCenter Version</span>
                <span>{{ appVersion }}</span>
              </div>
              <div>
                <span>User Email</span>
                <span>{{ userEmail }}</span>
              </div>
              <div>
                <span>Mail Kind</span>
                <span>{{ mailKind }}</span>
              </div>
            </div>
          </div>

          <h2><translate>Reports</translate></h2>

          <p v-if="reports.length == 0">
            <translate>No reports have been sent yet.</translate>
          </p>

          <div
            class="report"
            v-for="(report, reportIndex) in reports"
            :key="reportIndex"
          >
            <div>
              <span v-translate="{ report_id: report.id }">
                Report #%{report_id}
              </span>
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

import {
  getLatestReports,
  getUserInfo,
  getApplicationInfo,
  getSettings
} from "../lib/api.js";

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
      reportKindExplanation: "…",
      reportKindExplanationTitle: "…",

      // metadata sent with reports
      instanceID: "…",
      localIP: "…",
      postfixURL: "…",
      postfixVersion: "…",
      appVersion: "…",
      userEmail: "…",
      mailKind: "…",

      // translations
      InstanceIdText: this.$gettext(
        "A random identifier generated upon installation"
      )
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
      }
      if (seconds < 60 * 60) {
        let message = this.$gettext("%{m}m ago");
        return this.$gettextInterpolate(message, {
          m: Math.floor(seconds / 60)
        });
      }
      if (seconds < 60 * 60 * 24) {
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
        connection_stats_with_auth: this.$gettext(
          "IP addresses and timestamps of incoming SMTP connections that try to authenticate on ports 587 or 465. This data is obtained from the Postfix logs. It is used to identify and help prevent brute force attacks via the SMTP protocol based on automated threat sharing between Lightmeter users."
        ),
        insights_count: this.$gettext(
          "General statistics about recently generated Lightmeter insights. This data is used to improve the frequency of insights generation, e.g. avoiding too many notifications."
        ),
        log_lines_count: this.$gettext(
          "General non-identifiable information about which kinds of logs Postfix generates, and whether they are currently supported by Lightmeter ControlCenter. This is used to identify and prioritise support for unsupported log events."
        ),
        mail_activity: this.$gettext(
          "Statistics about the number of sent, bounced, deferred, and received messages. This helps identify Lightmeter performance issues and benchmark network health."
        ),
        top_domains: this.$gettext(
          "Domains that Postfix is managing locally. This is used to verify that reports are authentic."
        )
      };

      return explanations[report_kind] ? explanations[report_kind] : "…";
    }
  },
  mounted() {
    let vue = this;

    getUserInfo().then(function(userInfo) {
      vue.instanceID = userInfo.data.instance_id;
      vue.mailKind = userInfo.data.mail_kind;
      vue.postfixVersion = userInfo.data.postfix_version;
      vue.userEmail = userInfo.data.user.email;
    });
    getApplicationInfo().then(function(appInfo) {
      vue.appVersion = appInfo.data.version;
    });
    getSettings().then(function(response) {
      vue.localIP = response.data.general.postfix_public_ip;
      vue.postfixURL = response.data.general.public_url;
    });

    this.loadReports();
  },
  destroyed() {}
};
</script>

<style lang="less">
.metadata {
  display: inline-block;
  .metadata-table > div {
    display: flex;
    justify-content: space-between;
    span:first-child {
      font-weight: bold;
      margin-right: 1em;
    }
  }
}
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
