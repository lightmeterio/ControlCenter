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
        <ul class="status-list list-unstyled">
          <li
            v-for="(delivery, statusIndex) in result.entries"
            :key="statusIndex"
            :class="statusClass(delivery.status)"
            :title="statusTitle(delivery.status)"
          >
            {{ delivery.status }}
          </li>
          <li
            :class="statusClass('expired')"
            :title="statusTitle('expired')"
            v-show="isExpired(result)"
          >
            expired
          </li>
        </ul>

        <div
          v-show="showQueues"
          class="queue-name card-text"
          v-translate="{ queue: result.queue }"
        >
          Queue ID: %{queue}
        </div>

        <ul class="list-unstyled card-text">
          <li
            class="mt-3 card-text"
            v-for="(delivery, deliveryIndex) of result.entries"
            :key="deliveryIndex"
          >
            <!-- prettier-ignore -->
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
              %{attempts} delivery attempts %{status} with status code %{code} from %{begin} to %{end}
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

function emailDate(d) {
  return humanDateTime(d);
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
    }
  },
  methods: {
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
        returned: "status-returned"
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
        )
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
    },
    queueLabel(queue) {
      let message = this.$gettext("Queue ID: %{queue}");
      return this.$gettextInterpolate(message, { queue: queue });
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
  width: min-content;
  height: 100%;
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
</style>
