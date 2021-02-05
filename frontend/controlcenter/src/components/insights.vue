<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="insights" id="insights">
    <b-modal
      ref="modal-rbl-list"
      id="modal-rbl-list"
      size="lg"
      hide-footer
      centered
      :title="insightRblCheckedIpTitle"
    >
      <p class="intro">
        <!-- prettier-ignore -->
        <translate>These lists are recommending that your list be blocked &ndash; check their messages for hints</translate>
      </p>
      <span id="rbl-list-content">
        <div class="card" v-for="r of rbls" v-bind:key="r.text">
          <div class="card-body">
            <h5 class="card-title">
              <span class="badge badge-pill badge-warning">List</span
              >{{ r.rbl }}
            </h5>
            <p class="card-text">
              <span class="message-label">Message:</span>
              <span v-linkified:options="{target: {url: '_blank'}}">{{ r.text }}</span>
            </p>
          </div>
        </div>
      </span>

      <b-row class="vue-modal-footer">
        <b-col>
          <b-button
            class="btn-cancel"
            variant="outline-danger"
            @click="hideRBLListModal"
          >
            <!-- prettier-ignore -->
            <translate>Close</translate>
          </b-button>
        </b-col>
      </b-row>
    </b-modal>
    <b-modal
      ref="modal-msg-rbl"
      id="modal-msg-rbl"
      size="lg"
      hide-footer
      centered
      :title="insightMsgRblTitle"
    >
      <div class="modal-body">
        <blockquote>
          <span id="rbl-msg-rbl-content" v-linkified:options="{target: {url: '_blank'}}"> {{ msgRblDetails }} </span>
        </blockquote>
      </div>

      <b-row class="vue-modal-footer">
        <b-col>
          <b-button
            class="btn-cancel"
            variant="outline-danger"
            @click="hideRBLMsqModal"
          >
            <!-- prettier-ignore -->
            <translate>Close</translate>
          </b-button>
        </b-col>
      </b-row>

    </b-modal>
    <div
      v-for="insight of insightsTransformed"
      v-bind:key="insight.id"
      class="col-card col-md-6 h-25"
    >
      <div class="card">
        <div class="row">
          <div
            class="col-lg-1 col-md-2 col-sm-1 col-2 rating"
            v-bind:class="[insight.ratingClass]"
          ></div>
          <div class="col-lg-11 col-md-10 col-10">
            <div class="card-block">
              <div
                class="d-flex flex-row justify-content-between insight-header"
              >
                <p class="card-text category">{{ insight.category }}</p>

                <span
                  v-if="insight.help_link"
                  v-on:click="
                    onInsightInfo(
                      $event,
                      insight.help_link,
                      insight.content_type
                    )
                  "
                  v-b-tooltip.hover
                  :title="Info"
                >
                  <i class="fa fa-info-circle insight-help-button"></i>
                </span>
              </div>
              <h6 class="card-title title">{{ insight.title }}</h6>
              <p
                v-if="insight.content_type === 'high_bounce_rate'"
                class="card-text description"
                v-html="insight.description"
              ></p>

              <p
                v-if="insight.content_type === 'mail_inactivity'"
                class="card-text description"
                v-html="insight.description"
              ></p>
              <p
                v-if="insight.content_type === 'welcome_content'"
                class="card-text description"
              >
                <!-- prettier-ignore -->
                <translate>Insights reveal mailops problems in real time &ndash; both here, and via notifications</translate
                >
              </p>
              <p
                v-if="insight.content_type === 'insights_introduction_content'"
                class="card-text description"
              >
                <!-- prettier-ignore -->
                <translate>Join us on the journey to better mailops! We're listening for your feedback</translate>
              </p>
              <p
                v-if="insight.content_type === 'local_rbl_check'"
                class="card-text description"
              >
                <span v-html="insight.description.message"></span>

                <button
                  v-b-modal.modal-rbl-list
                  v-on:click="onBuildInsightRbl(insight.id)"
                  class="btn btn-sm"
                >
                  <!-- prettier-ignore -->
                  <translate>Details</translate>
                </button>
              </p>
              <p
                v-if="insight.content_type === 'message_rbl'"
                class="card-text description"
              >
                <span v-html="insight.description.message"></span>
                <button
                  v-b-modal.modal-msg-rbl
                  v-on:click="onBuildInsightMsgRbl(insight.id)"
                  class="btn btn-sm"
                >
                  <!-- prettier-ignore -->
                  <translate>Details</translate>
                </button>
              </p>
              <p class="card-text time">{{ insight.modTime }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import moment from "moment";
import { getApplicationInfo } from "@/lib/api";
import tracking from "../mixin/global_shared.js";
import linkify from 'vue-linkify';
import Vue from "vue";

Vue.directive('linkified', linkify);

export default {
  name: "insights",
  mixins: [tracking],
  props: {
    insights: Array
  },
  computed: {
    insightsTransformed() {
      return this.transformInsights(this.insights);
    },
    Info() {
      return this.$gettext("Info");
    }
  },
  mounted() {
    let vue = this;
    getApplicationInfo().then(function(response) {
      vue.applicationData = response.data;
    });
  },
  data() {
    return {
      rbls: [],
      msgRblDetails: "",
      insightRblCheckedIpTitle: "",
      insightMsgRblTitle: "",
      applicationData: { version: "" }
    };
  },
  methods: {
    onBuildInsightRbl: function(id) {
      this.buildInsightRblCheckedIp(id);
      this.buildInsightRblList(id);
      this.trackEvent("InsightDescription", "openRblModal");
    },
    onBuildInsightMsgRbl(id) {
      this.buildInsightMsgRblTitle(id);
      this.buildInsightMsgRblDetails(id);
      this.trackEvent("InsightDescription", "openHostBlockModal");
    },
    high_bounce_rate_title() {
      return this.$gettext("High Bounce Rate");
    },
    mail_inactivity_title() {
      return this.$gettext("Mail Inactivity");
    },
    welcome_content_title() {
      return this.$gettext("Your first Insight");
    },
    insights_introduction_content_title() {
      let translation = this.$gettext("Welcome to Lightmeter %{version}");
      return this.$gettextInterpolate(translation, {version: this.applicationData.version});
    },
    local_rbl_check_title() {
      return this.$gettext("IP on shared blocklist");
    },
    message_rbl_title(i) {
      let translation = this.$gettext("IP blocked by %{host}");
      return this.$gettextInterpolate(translation, {host: i.content.host});
    },
    high_bounce_rate_description(i) {
      let c = i.content;
      let translation = this.$gettext("<b>%{bounceValue}%</b> bounce rate between %{intFrom} and %{intTo}");

      return this.$gettextInterpolate(translation, {
        bounceValue: c.value * 100,
        intFrom: formatInsightDescriptionDateTime(c.interval.from),
        intTo:formatInsightDescriptionDateTime(c.interval.to)
      });
    },
    mail_inactivity_description(i) {
      let c = i.content;
      let translation = this.$gettext("No emails were sent between %{intFrom} and %{intTo}");

      return this.$gettextInterpolate(translation, {
        intFrom: formatInsightDescriptionDateTime(c.interval.from),
        intTo:formatInsightDescriptionDateTime(c.interval.to)
      });
    },
    local_rbl_check_description(i) {
      let c = i.content;
      // TODO: handle difference between singular (one RBL) and plurals
      let translation = this.$gettext("The IP address %{ip} is listed by <strong>%{rblCount}</strong> <abbr title='Real-time Blackhole List'>RBL</abbr>s" );

      let message = this.$gettextInterpolate(translation, {ip: c.address, rblCount: c.rbls.length});

      return {
        id: i.id.toString(),
        message: message
      };
    },
    message_rbl_description(i) {
      let c = i.content;
      let translation = this.$gettext("The IP %{ip} cannot deliver to %{recipient} (<strong>%{host}</strong>)");

      let message = this.$gettextInterpolate(translation, {ip: c.address, recipient: c.recipient, host: c.host});

      return {
        id: i.id.toString(),
        message: message
      };
    },
    transformInsights(insights) {
      let vue = this;
      if (insights === null) {
        return;
      }

      let insightsTransformed = [];
      for (let insight of insights) {
        insight.category = vue.buildInsightCategory(insight);
        insight.modTime = vue.buildInsightTime(insight);
        insight.title = vue.buildInsightTitle(insight);
        insight.ratingClass = vue.buildInsightRating(insight);
        insight.description = vue.buildInsightDescriptionValues(insight);
        insightsTransformed.push(insight);
      }

      return insightsTransformed;
    },
    buildInsightCategory(insight) {
      // FIXME We shouldn't capitalise in the code -- leave that for the i18n workflow to decide
      return (
        insight.category.charAt(0).toUpperCase() + insight.category.slice(1)
      );
    },
    buildInsightTime(insight) {
      return moment(insight.time).format("DD MMM YYYY | h:mmA");
    },
    buildInsightTitle(insight) {
      const s = this[insight.content_type + "_title"];

      if (typeof s == "string") {
        return s;
      }

      if (typeof s == "function") {
        return s(insight);
      }

      let translation = this.$gettext("Title for %{content}")

      return this.$gettextInterpolate(translation, {content: insight.content_type})
    },
    buildInsightRating(insight) {
      return insight.rating;
    },
    buildInsightDescriptionValues(insight) {
      let handler = this[insight.content_type + "_description"];

      if (handler === undefined) {
        // NOTE: this string is for debug purposes only, therefore does not need to be translated
        // It happens only during the development of a new insight, as a final one should always have a description
        return "Description for " + insight.content_type;
      }

      return handler(insight);
    },
    buildInsightRblCheckedIp(insightId) {
      let insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return "";
      }

      let translation = this.$gettext("RBLS for %{address}");

      this.insightRblCheckedIpTitle =
        this.$gettextInterpolate(translation, {address: insight.content.address});
    },
    buildInsightRblList(insightId) {
      let insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return;
      }

      this.rbls = insight.content.rbls;
    },
    buildInsightMsgRblDetails(insightId) {
      let insight = this.insights.find(i => i.id == insightId);

      if (insight === undefined) {
        return;
      }

      this.msgRblDetails = insight.content.message;
    },
    buildInsightMsgRblTitle(insightId) {
      let insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return "";
      }

      let translation = this.$gettext("Original response from %{recipient} (%{host})")

      this.insightMsgRblTitle = this.$gettextInterpolate(translation, {recipient: insight.content.recipient, host: insight.content.host});
    },
    onInsightInfo(event, helpLink, contentType) {
      event.preventDefault();

      this.trackEventArray("InsightsInfoButton", [
        "click",
        helpLink,
        contentType
      ]);

      window.open(helpLink);
    },
    hideRBLListModal() {
      this.$refs["modal-rbl-list"].hide();
    },
    hideRBLMsqModal() {
      this.$refs["modal-msg-rbl"].hide();
    }
  }
};

function formatInsightDescriptionDateTime(d) {
  // TODO: this should be formatted according to the chosen language
  return moment(d).format("DD MMM. (h:mmA)");
}
</script>
<style>
.insights {
  margin-bottom: 1em;
  margin-top: 0.9rem;
  align-content: start;
}

.insights .col-card {
  margin-top: 1rem;
  align-content: start;
}

.insights .card {
  background: #ffffff 0% 0% no-repeat padding-box;
  box-shadow: 0px 0px 6px #0000001a;
  border: 1px solid #e6e7e7;
  border-radius: 5px;
}
.insights .card-text.category {
  background-color: #f2f2f2;
  border-radius: 18px;
  padding: 0.4em 1.5em;
  margin-bottom: 1.2em;
  width: min-content;
  height: 100%;
}
.insights .card-text.category,
.insights .card-text.time {
  color: #414445;
  font-size: 10px;
  font-weight: bold;
}
.insights .card-block {
  padding: 0.5rem 0.4rem 0.5rem 0rem;
  text-align: left;
}
.insights .card .rating {
  background-clip: content-box;
  background-color: #f0f8fc;
}
.insights .card-text {
  font: 12px/18px "Open Sans", sans-serif;
}
.insights .card-text.time {
  background: #f0f8fc 0% 0% no-repeat padding-box;
  border-radius: 5px;
  padding: 1.5em;
}
.insights .card-title {
  margin: 0 0 0.3rem 0;
  font: 18px Inter;
  font-weight: bold;
  color: #202324;
  letter-spacing: 0px;
}
.insights .card .rating {
  background-clip: content-box;
  background-color: #f0f8fc;
}
.insights .card .rating.bad {
  background: rgb(255, 92, 111);
  background: linear-gradient(
      0deg,
      rgba(255, 92, 111, 1) 85%,
      rgba(230, 231, 231, 1) 85%
    )
    content-box;
}
.insights .card .rating.ok {
  background: rgb(255, 220, 0);
  background: linear-gradient(
      0deg,
      rgba(255, 220, 0, 1) 50%,
      rgba(230, 231, 231, 1) 50%
    )
    content-box;
}
.insights .card .rating.good {
  background: rgb(135, 197, 40);
  background: linear-gradient(
      0deg,
      rgba(135, 197, 40, 1) 15%,
      rgba(230, 231, 231, 1) 15%
    )
    content-box;
}
.insights .card svg {
  margin-right: 0.05em;
}
.insights svg.insight-help-button {
  font-size: 1.3em;
}

svg.insight-help-button:hover {
  color: #2c9cd6;
}
svg.insight-help-button {
  color: #c5c7c6;
}

.insights .card-text.description button {
  padding: 0.4em 1.5em;
  margin-bottom: 0;
  background: #f9f9f9 0% 0% no-repeat padding-box;
  border-radius: 10px;
  font-size: 10px;
  font-weight: bold;
  color: #1d8caf;
  line-height: 1;
  border: 1px solid #e6e7e7;
  margin-left: 0.25em;
}

.insights .card-text.description button:hover {
  background-color: #1d8caf;
  color: #ffffff;
}


#modal-msg-rbl blockquote {
  font-style: italic;
  color: #555555;
  padding: 1em 30px 1em 0.6em;
  border-left: 8px solid #1d8caf;
  line-height: 1.3;
  position: relative;
  background: #f9f9f9;
}

#modal-msg-rbl blockquote::before {
  font-family: Inter;
  content: "\201C";
  color: #daebf4;
  font-size: 3em;
  position: absolute;
  left: 10px;
  top: -10px;
}

#modal-msg-rbl blockquote::after {
  content: "";
}

#modal-msg-rbl blockquote span {
  display: block;
  color: #333333;
  font-style: normal;
  margin-top: 1em;
  font-size: 0.7em;
}

#modal-msg-rbl .btn-cancel,
#modal-rbl-list .btn-cancel {
  background: #ff5c6f33 0% 0% no-repeat padding-box;
  border: 1px solid #ff5c6f;
  border-radius: 2px;
  opacity: 0.8;
  text-align: center;
  font: normal normal bold 14px/24px Open Sans;
  letter-spacing: 0px;
  color: #820d1b;
}

#modal-msg-rbl .btn-cancel:hover,
#modal-rbl-list .btn-cancel:hover {
  color: #212529;
  text-decoration: none;
}

#modal-rbl-list .modal-body .intro {
  font-size: 0.7em;
}

#rbl-list-content .card {
  margin-bottom: 0.8em;
  background-color: #f9f9f9;
  border: 1px solid #c5c7c6;
}
.modal-content .card .card-title {
  font-size: 19px;
  font-weight: bold;
  color: #202324;
  font-family: Inter;
}

.modal-content .card .card-text {
  font-size: 15px;
}

#rbl-list-content .card .message-label {
  color: #7a82ab;
  font-weight: bold;
  padding-right: 0.5em;
  font-size: 15px;
}

#rbl-list-content .badge {
  font-size: 80%;
}

#rbl-list-content .badge-info {
  background-color: #fce4c4;
  color: #ff5c6f;
}

#rbl-list-content .badge-warning {
  background-color: #ff5c6f;
  color: white;
  margin-right: 0.6em;
}

.modal-content {
  border-radius: 0;
}

@media (max-width: 768px) {
  .modal-content {
    width: 100%;
  }
}

/* Note: workaround for vue bootstraps weird default modal footer handling */
.vue-modal-footer {
  padding-top: .75rem;
  border-top: 1px solid #dee2e6;
  text-align: right;
  margin-top: 1em;
}
</style>
