<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div id="signals-page" class="d-flex flex-column min-vh-100">
    <mainheader></mainheader>
    <div class="container main-content">
      <div class="row">
        <div class="col-md-12">
          <h2>
            <translate>Latest 20 signals shared with the network</translate>
          </h2>

          <div class="metadata">
            <p>The following metadata is sent along with signals</p>
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
            </div>
          </div>

          <h2><translate>Signals</translate></h2>

          <p v-if="signals.length == 0">
            <translate>No signals have been sent yet.</translate>
          </p>

          <div
            class="signal"
            v-for="(signal, signalIndex) in signals"
            :key="signalIndex"
          >
            <div>
              <span v-translate="{ signal_id: signal.id }">
                Signal #%{signal_id}
              </span>
              –
              <span :title="signal.dispatch_time">{{
                humanDuration(signal.dispatch_time)
              }}</span>
              –
              <button
                class="btn btn-sm btn-info"
                v-b-modal.modal-explanation
                v-on:click="
                  signalKindExplanationTitle = signal.kind;
                  signalKindExplanation = explanation(signal.kind);
                "
              >
                <span>{{ signal.kind }}</span>
                <i class="far fa-question-circle signal_help_btn"></i>
              </button>
            </div>
            <vue-json-pretty :data="signal.value"> </vue-json-pretty>
          </div>

          <b-modal
            ref="modal-explanation"
            id="modal-explanation"
            hide-footer
            :title="signalKindExplanationTitle"
            cancel-only
          >
            {{ signalKindExplanation }}
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
  getLatestSignals,
  getUserInfo,
  getApplicationInfo,
  getSettings
} from "../lib/api.js";

import tracking from "../mixin/global_shared.js";
import auth from "../mixin/auth.js";

import VueJsonPretty from "vue-json-pretty";
import "vue-json-pretty/lib/styles.css";

export default {
  name: "signals",
  components: {
    VueJsonPretty
  },
  mixins: [tracking, auth],
  data() {
    return {
      signals: [],
      signalKindExplanation: "…",
      signalKindExplanationTitle: "…",

      // metadata sent with signals
      instanceID: "…",
      localIP: "…",
      postfixURL: "…",
      postfixVersion: "…",
      appVersion: "…",
      userEmail: "…",

      // translations
      InstanceIdText: this.$gettext(
        "A random identifier generated upon installation"
      )
    };
  },
  computed: {},
  methods: {
    loadSignals() {
      let vue = this;

      getLatestSignals().then(function(response) {
        vue.signals = response.data;
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
    explanation(signal_kind) {
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
          "Domains that Postfix is managing locally. This is used to verify that signals are authentic."
        )
      };

      return explanations[signal_kind] ? explanations[signal_kind] : "…";
    }
  },
  mounted() {
    let vue = this;

    getUserInfo().then(function(userInfo) {
      vue.instanceID = userInfo.data.instance_id;
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

    this.loadSignals();
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
.signal {
  margin: auto;
  margin-top: 1rem;
  padding-top: 1rem;

  & + .signal {
    border-top: 1px dotted #bdc3c7;
  }

  .vjs-tree {
    display: inline-block;
  }
}

.signal_help_btn {
  margin-left: 10px;
}
</style>
