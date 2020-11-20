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
          >Close</b-button
        >
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
          >Close</b-button
        >
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
                  title="Info"
                >
                  <i class="fa fa-info-circle insight-help-button"></i>
                </span>
              </div>
              <h6 class="card-title title">{{ insight.title }}</h6>
              <p
                v-if="insight.content_type === 'high_bounce_rate'"
                class="card-text description"
              >
                <b>{{ insight.description.value }}</b> bounce rate between
                {{ insight.description.from }} and {{ insight.description.to }}
              </p>
              <p
                v-if="insight.content_type === 'mail_inactivity'"
                class="card-text description"
              >
                No emails were sent between
                {{ insight.description.from }} and {{ insight.description.to }}
              </p>
              <p
                v-if="insight.content_type === 'welcome_content'"
                class="card-text description"
              >
                Insights are time-based analyses relating to your mailserver
              </p>
              <p
                v-if="insight.content_type === 'insights_introduction_content'"
                class="card-text description"
              >
                Keep Control Center running to generate Insights; checks run
                every few seconds
              </p>
              <p
                v-if="insight.content_type === 'local_rbl_check'"
                class="card-text description"
              >
                The IP address {{ insight.description.address }} is listed by
                <strong>{{ insight.description.length }} </strong>
                <abbr title="Real-time Blackhole List">RBL</abbr>s

                <button
                  v-b-modal.modal-rbl-list
                  v-on:click="
                    buildInsightRblCheckedIp(insight.id);
                    buildInsightRblList(insight.id);
                  "
                  class="btn btn-sm"
                >
                  Details
                </button>
              </p>
              <p
                v-if="insight.content_type === 'message_rbl'"
                class="card-text description"
              >
                The IP {{ insight.description.address }} cannot deliver to
                {{ insight.description.recipient }} (<strong
                  >{{ insight.description.host }} </strong
                >)
                <button
                  v-b-modal.modal-msg-rbl
                  v-on:click="
                    buildInsightMsgRblTitle(insight.id);
                    buildInsightMsgRblDetails(insight.id);
                  "
                  class="btn btn-sm"
                >
                  Details
                </button>
              </p>
              <p class="card-text time">{{ insight.time }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import moment from "moment";

export default {
  name: "insights",
  props: {
    insights: Array
  },
  computed: {
    insightsTransformed() {
      return this.transformInsights(this.insights);
    }
  },
  data() {
    return {
      rbls: [],
      msgRblDetails: "",
      insightRblCheckedIpTitle: "",
      insightMsgRblTitle: "",
      insightsTitles: {
        high_bounce_rate: "High Bounce Rate", //todo translate
        mail_inactivity: "Mail Inactivity", //todo translate
        welcome_content: "Your first Insight", //todo translate
        insights_introduction_content: "Breaking new ground", //todo translate
        local_rbl_check: "IP on shared blocklist", //todo translate
        message_rbl: i => {
          return "IP blocked by " + i.content.host;
        } //todo translate
      },
      insightsDescriptions: {
        high_bounce_rate: function(i) {
          let c = i.content;
          return {
            value: c.value * 100,
            from: formatInsightDescriptionDateTime(c.interval.from),
            to: formatInsightDescriptionDateTime(c.interval.to)
          };
        },
        // "{{ translate `<b>%i%%</b> bounce rate between %s and %s`}}"
        mail_inactivity: function(i) {
          let c = i.content;
          return {
            from: formatInsightDescriptionDateTime(c.interval.from),
            to: formatInsightDescriptionDateTime(c.interval.to)
          };
        },
        // "{{ translate `Keep Control Center running to generate Insights; checks run every few seconds` }}"
        local_rbl_check: function(i) {
          let c = i.content;

          return {
            address: c.address,
            length: c.rbls.length,
            id: i.id.toString()
          };
        },
        //{{ translate `Details` }}
        // _paq.push(['trackEvent', 'InsightDescription', 'openRblModal']) -> onclick rblListModal
        message_rbl: function(i) {
          let c = i.content;

          return {
            address: c.address,
            recipient: c.recipient,
            host: c.host,
            id: i.id.toString()
          };
          // "{{ translate `The IP %s cannot deliver to %s (<strong>%s</strong>)` }}.
        }
        // {{ translate `Details` }}
        // _paq.push(['trackEvent', 'InsightDescription', 'openHostBlockModal']); -> on click msg_rbl_list_modal
      }
    };
  },
  methods: {
    transformInsights(insights) {
      let vue = this;
      if (insights === null) {
        return;
      }

      let insightsTransformed = [];
      for (let insight of insights) {
        insight.category = vue.buildInsightCategory(insight);
        insight.time = vue.buildInsightTime(insight);
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
      const s = this.insightsTitles[insight.content_type];

      if (typeof s == "string") {
        return s;
      }

      if (typeof s == "function") {
        return s(insight);
      }

      return "Title for " + insight.content_type;
    },
    buildInsightRating(insight) {
      return insight.rating;
    },
    buildInsightDescriptionValues(insight) {
      let handler = this.insightsDescriptions[insight.content_type];

      if (handler === undefined) {
        return "Description for " + insight.content_type;
      }

      return handler(insight);
    },
    buildInsightRblCheckedIp(insightId) {
      let insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return "";
      }
      this.insightRblCheckedIpTitle = "RBLS for" + insight.content.address;
    },
    buildInsightRblList(insightId) {
      let insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return;
      }

      this.rbls = insight.content.rbls;
    },
    buildInsightMsgRblDetails(insightId) {
      var insight = this.insights.find(i => i.id == insightId);

      if (insight === undefined) {
        return;
      }

      this.msgRblDetails = insight.content.message;
    },
    buildInsightMsgRblTitle(insightId) {
      var insight = this.insights.find(i => i.id === insightId);

      if (insight === undefined) {
        return "";
      }

      this.insightMsgRblTitle =
        "Original response from " +
        insight.content.recipient +
        " (" +
        insight.content.host +
        ")";
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
