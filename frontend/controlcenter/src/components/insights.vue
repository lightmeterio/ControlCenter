<template>
  <div class="insights" id="insights">
    <b-modal
      ref="modal-rbl-list"
      id="modal-rbl-list"
      hide-footer
      :title="insightRblCheckedIpTitle"
    >
      <div class="modal-body">
        <span id="rbl-list-content">
          <ul>
            <li v-for="r of rbls" v-bind:key="r.text">
              <b>{{ r.rbl }}</b
              >{{ r.text }}
            </li>
          </ul>
        </span>
      </div>
      <div class="modal-footer">
        <b-button
          class="btn-cancel"
          variant="outline-danger"
          @click="hideRBLListModal"
        >
          <!-- prettier-ignore -->
          <translate>Close</translate>
        </b-button>
      </div>
    </b-modal>
    <b-modal
      ref="modal-msg-rbl"
      id="modal-msg-rbl"
      hide-footer
      :title="insightMsgRblTitle"
    >
      <div class="modal-body">
        <span id="rbl-msg-rbl-content"> {{ msgRblDetails }} </span>
      </div>
      <div class="modal-footer">
        <b-button
          class="btn-cancel"
          variant="outline-danger"
          @click="hideRBLMsqModal"
        >
          <!-- prettier-ignore -->
          <translate>Close</translate>
        </b-button>
      </div>
    </b-modal>
    <div
      v-for="insight of insightsTransformed"
      v-bind:key="insight.id"
      class="col-md-6"
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
                <translate>Insights are time-based analyses relating to your mailserver</translate
                >
              </p>
              <p
                v-if="insight.content_type === 'insights_introduction_content'"
                class="card-text description"
              >
                <!-- prettier-ignore -->
                <translate>Keep Control Center running to generate Insights; checks run every few seconds</translate>
              </p>
              <p
                v-if="insight.content_type === 'local_rbl_check'"
                class="card-text description"
                v-html="insight.description.message"
              >
                <button
                  v-b-modal.modal-rbl-list
                  v-on:click="
                    buildInsightRblCheckedIp(insight.id);
                    buildInsightRblList(insight.id);
                  "
                  class="btn btn-sm"
                >
                  <!-- prettier-ignore -->
                  <translate>Details</translate>
                </button>
              </p>
              <p
                v-if="insight.content_type === 'message_rbl'"
                class="card-text description"
                v-html="insight.description.message"
              >
                <button
                  v-b-modal.modal-msg-rbl
                  v-on:click="
                    buildInsightMsgRblTitle(insight.id);
                    buildInsightMsgRblDetails(insight.id);
                  "
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
import { sprintf } from "sprintf-js";

export default {
  name: "insights",
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
  data() {
    return {
      rbls: [],
      msgRblDetails: "",
      insightRblCheckedIpTitle: "",
      insightMsgRblTitle: ""
    };
  },
  methods: {
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
      return this.$gettext("Breaking new ground");
    },
    local_rbl_check_title() {
      return this.$gettext("IP on shared blocklist");
    },
    message_rbl_title(i) {
      return this.$gettext("IP blocked by ") + i.content.host;
    },
    high_bounce_rate_description(i) {
      let c = i.content;
      let translation = this.$gettext(
        "<b>%i%%</b> bounce rate between %s and %s"
      );
      return sprintf(
        translation,
        c.value * 100,
        formatInsightDescriptionDateTime(c.interval.from),
        formatInsightDescriptionDateTime(c.interval.to)
      );
    },
    mail_inactivity_description(i) {
      let c = i.content;
      let translation = this.$gettext("No emails were sent between %s and %s");
      return sprintf(
        translation,
        formatInsightDescriptionDateTime(c.interval.from),
        formatInsightDescriptionDateTime(c.interval.to)
      );
    },
    local_rbl_check_description(i) {
      let c = i.content;
      let translation = this.$gettext(
        "The IP address %s is listed by <strong>%d</strong> <abbr title='Real-time Blackhole List'>RBL</abbr>s"
      );
      let message = sprintf(translation, c.address, c.rbls.length);
      return {
        id: i.id.toString(),
        message: message
      };
    },
    // _paq.push(['trackEvent', 'InsightDescription', 'openRblModal']) -> onclick rblListModal
    message_rbl_description(i) {
      let c = i.content;
      let translation = this.$gettext(
        "The IP %s cannot deliver to %s (<strong>%s</strong>)"
      );

      let message = sprintf(translation, c.address, c.recipient, c.host);

      return {
        id: i.id.toString(),
        message: message
      };
    },
    // _paq.push(['trackEvent', 'InsightDescription', 'openHostBlockModal']); -> on click msg_rbl_list_modal
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

      return this.$gettext("Title for ") + insight.content_type;
    },
    buildInsightRating(insight) {
      return insight.rating;
    },
    buildInsightDescriptionValues(insight) {
      let handler = this[insight.content_type + "_description"];

      if (handler === undefined) {
        return this.$gettext("Description for ") + insight.content_type;
      }

      return handler(insight);
    },
    buildInsightRblCheckedIp(insightId) {
      let insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return "";
      }
      this.insightRblCheckedIpTitle =
        this.$gettext("RBLS for") + insight.content.address;
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

      this.insightMsgRblTitle = sprintf(
        this.$gettext("Original response from %s (%s)"),
        insight.content.recipient,
        insight.content.host
      );
    },
    onInsightInfo(event, helpLink, contentType) {
      event.preventDefault();
      /*
      _paq.push([
        "trackEvent",
        "InsightsInfoButton",
        "click",
        helpLink,
        contentType
      ]);
       */
      console.log(contentType); // todo remove the consol.log after tracking is enabled
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
  return moment(d).format("DD MMM. (h:mmA)");
}
</script>
<style>
.insights {
  margin-bottom: 1em;
}

.insights .card {
  background: #ffffff 0% 0% no-repeat padding-box;
  box-shadow: 0px 0px 6px #0000001a;
  border: 1px solid #e6e7e7;
  border-radius: 5px;
  margin-top: 0.9rem;
}
.insights .card-text.category {
  background-color: #f2f2f2;
  border-radius: 18px;
  padding: 0.4em 1.5em;
  margin-bottom: 0em;
  width: min-content;
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
  margin-right: 0.8em;
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
  margin-left: 0.2em;
}

.insights .card-text.description button:hover {
  background-color: #1d8caf;
  color: #ffffff;
}
</style>
