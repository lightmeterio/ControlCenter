<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <b-container class="results mt-4">
    <ul
      v-for="(result, resultIndex) in results"
      :key="resultIndex"
      class="detective-result-cell card list-unstyled"
    >
      <li class="card-body">
        <div class="card-status-logs">
          <ul class="status-list list-unstyled">
            <li
              v-for="(delivery, statusIndex) in result.entries"
              :key="statusIndex"
              :class="statusClass(delivery.status)"
              v-b-tooltip.hover
              :title="statusTitle(delivery.status)"
            >
              {{ delivery.status }}
            </li>
            <li
              :class="statusClass('expired')"
              v-b-tooltip.hover
              :title="statusTitle('expired')"
              v-show="isExpired(result)"
            >
              expired
            </li>
          </ul>

          <b-button
            v-on:click="downloadRawLogsInInterval(result)"
            variant="primary"
            size="sm"
            style="margin-left: 1rem;"
            v-b-tooltip.hover
            :title="titleDownloadLogsAroundDelivery"
          >
            <i class="fas fa-download"></i>
            <translate>Logs</translate>
          </b-button>
        </div>

        <div
          v-show="showQueues"
          class="queue-name card-text"
          v-translate="{ queue: result.queue, mid: result.message_id }"
        >
          Queue ID: %{queue} / Message ID: %{mid}
        </div>

        <div v-show="showFromTo" class="card-text">
          {{ result.entries[0].from }} → {{ result.entries[0].to.join(", ") }}
        </div>

        <ul class="list-unstyled card-text">
          <li
            class="mt-3 card-text"
            v-for="(delivery, deliveryIndex) of result.entries"
            :key="deliveryIndex"
          >
            <span
              v-show="hasMultipleDeliveryAttempts(delivery)"
              render-html="true"
              v-translate="{
                attempts: formatAttempts(delivery),
                status: formatMultipleStatus(delivery),
                code: formatMultipleDsn(delivery),
                begin: formatMinTime(delivery),
                end: formatMaxTime(delivery)
              }"
              class="mt-3 card-text"
            >
              %{attempts} delivery attempts %{status} with status code %{code}
              from %{begin} to %{end}
            </span>

            <span
              v-show="!hasMultipleDeliveryAttempts(delivery)"
              render-html="true"
              v-translate="{
                status: formatMultipleStatus(delivery),
                code: formatMultipleDsn(delivery),
                time: formatMinTime(delivery)
              }"
              class="mt-3 card-text"
            >
              Message %{status} with status code %{code} at %{time}
            </span>
            <span class="relays" v-b-tooltip.hover :title="titleRelay">
              ({{ delivery.relays.join(", ") }})
            </span>
            <ul
              v-show="delivery.log_msgs != null && delivery.log_msgs.length > 0"
              class="list-unstyled"
            >
              <li
                v-for="(logMsg, logIndex) of delivery.log_msgs"
                :key="logIndex"
                class="raw-log card em small"
              >
                {{ logMsg }}
              </li>
            </ul>
          </li>
        </ul>
      </li>
    </ul>

    <div v-show="showStatusCodeMoreInfo" render-html="true" v-translate>
      More on status codes: %{openLink}IANA's reference list%{closeLink}
    </div>
  </b-container>
</template>

<script>
import { humanDateTime } from "@/lib/date.js";
import tracking from "@/mixin/global_shared.js";
import moment from "moment";

import { linkToRawLogsInInterval } from "@/lib/api.js";

function emailDate(d) {
  return humanDateTime(d);
}

function formatTimeWithOffsetInSeconds(t, offsetInSeconds) {
  return moment
    .utc(t)
    .add(offsetInSeconds, "s")
    .format("YYYY-MM-DD HH:mm:ss");
}

export default {
  mixins: [tracking],
  props: {
    results: {
      type: Array,
      default: null
    },
    showQueues: {
      type: Boolean,
      default: false
    },
    showFromTo: {
      type: Boolean,
      default: false
    }
  },
  methods: {
    downloadRawLogsInInterval(result) {
      let from = formatTimeWithOffsetInSeconds(result.entries[0].time_min, -10);
      let to = formatTimeWithOffsetInSeconds(
        result.entries[result.entries.length - 1].time_max,
        +5
      );

      let link = linkToRawLogsInInterval(from, to, "plain", "inline");

      window.open(link);
    },
    hasMultipleDeliveryAttempts(delivery) {
      return delivery.number_of_attempts > 1;
    },
    statusClass: function(status) {
      let baseClass = "delivery-status card-text ";

      let customClass = {
        sent: "status-sent",
        bounced: "status-bounced",
        deferred: "status-deferred",
        expired: "status-expired",
        returned: "status-returned",
        received: "status-received"
      }[status];

      return baseClass + customClass;
    },
    statusTitle: function(status) {
      return {
        sent: this.$gettext("Message successfully sent"),
        bounced: this.$gettext("Message refused by recipient's mail provider"),
        deferred: this.$gettext("Message temporarily refused and retried"),
        expired: this.$gettext(
          "Message delivery abandoned after too many deferred attempts"
        ),
        returned: this.$gettext(
          "Return notification sent back to original sender"
        ),
        received: this.$gettext("Received message")
      }[status];
    },
    isExpired: function(result) {
      return result.entries.reduce((a, r) => a || r.expired, false);
    },
    formatTime(time) {
      return (
        `<span class="detective-result-time">` + emailDate(time) + "</span>"
      );
    },
    formatSingleTime(result) {
      return this.formatTime(result.entries[0].time_min);
    },
    formatSingleStatus(result) {
      return (
        `<span class="detective-result-status">` +
        result.entries[0].status +
        "</span>"
      );
    },
    formatSingleDsn(result) {
      return (
        `<span class="detective-result-dsn">` +
        result.entries[0].dsn +
        "</span>"
      );
    },
    formatMultipleStatus(delivery) {
      return (
        `<span class="detective-result-status">` + delivery.status + "</span>"
      );
    },
    formatMultipleDsn(delivery) {
      return `<span class="detective-result-dsn">` + delivery.dsn + "</span>";
    },
    formatMinTime(delivery) {
      return this.formatTime(delivery.time_min);
    },
    formatAttempts(delivery) {
      return (
        `<span class="detective-result-attempts">` +
        delivery.number_of_attempts +
        "</span>"
      );
    },
    formatMaxTime(delivery) {
      return this.formatTime(delivery.time_max);
    }
  },
  data() {
    return {
      openLink: `<a href="https://www.iana.org/assignments/smtp-enhanced-status-codes/smtp-enhanced-status-codes.xhtml">`,
      closeLink: `</a>`
    };
  },
  computed: {
    showStatusCodeMoreInfo() {
      return this.results != null && this.results.length > 0;
    },
    titleDownloadLogsAroundDelivery: function() {
      return this.$gettext(
        "View mail server logs around this delivery (-10s +5s)"
      );
    },
    titleRelay: function() {
      return this.$gettext("Message was sent to this server (relay)");
    }
  }
};
</script>

<style lang="less">
.delivery-status {
  text-transform: capitalize;
  border-radius: 18px;
  padding: 0.4em 1.5em;
  margin-right: 0.5em;
  margin-bottom: 0.5em;
  font-weight: bold;
  color: #724141;
}

.detective-result-cell {
  margin-bottom: 0.8em;
  background-color: #f9f9f9;
  border: 1px solid #c5c7c6;
}

.status-list {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  justify-content: flex-start;
}

.detective-result-status {
  font-weight: bold;
}

.detective-result-dsn {
  background-color: #e7e8f0;
  border-radius: 10px;
  font-weight: bold;
  padding-left: 5px;
  padding-right: 5px;
}

.card-status-logs {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.detective-result-attempts {
  font-weight: bold;
}

.detective-result-time {
  font-weight: bold;
  color: #7d7d7d;
}

.queue-name {
  font-weight: bold;
  color: #707070;
  margin-top: 1em;
}

.status-bounced {
  background-color: #fed0d0;
}

.status-expired {
  background-color: #fed0d0;
}

.status-deferred {
  background-color: #faf083;
}

.status-returned {
  background-color: #7faafa;
}

.status-sent {
  background-color: #8cfa86;
}

.status-received {
  background-color: #ecf0f1;
}

.relays {
  color: #7f8c8d;
}

.raw-log {
  margin: 5px;
  padding: 5px;
}
</style>
