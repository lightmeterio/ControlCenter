<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <b-container class="results mt-4">
    <div v-for="(result, index) in results" :key="index">
      <h3>
        <!-- prettier-ignore -->
        <translate>Message Status</translate>:
        <span
          v-for="delivery in result"
          :key="delivery.time_min"
          :class="statusClass(delivery.status)"
          >{{ delivery.status }}</span
        >
      </h3>

      <template v-if="hasOnlyOneDelivery(result)">
        <p class="mt-3">
          {{ emailDate(result[0].time_min) }}
        </p>
        <p>
          <!-- prettier-ignore -->
          <translate>Status code</translate>:
          {{ result[0].dsn }}
        </p>
      </template>
      <div v-else v-for="delivery in result" :key="delivery.time_min">
        <p class="mt-3">
          {{ delivery.number_of_attempts }}
          <!-- prettier-ignore -->
          <translate>delivery attempt(s)</translate>
          <span :class="statusClass(delivery.status)">
            {{ delivery.status }}
          </span>
          <!-- prettier-ignore -->
          <translate>with status code</translate>
          {{ delivery.dsn }}
          -
          <span class="text-secondary">
            {{ emailDate(delivery.time_min) }}
          </span>
          <template v-if="!(delivery.time_max == delivery.time_min)">
            -
            <span class="text-secondary">
              {{ emailDate(delivery.time_max) }}
            </span>
          </template>
        </p>
      </div>
      <p>
        (You can find more information about status codes on
        <a
          href="https://www.iana.org/assignments/smtp-enhanced-status-codes/smtp-enhanced-status-codes.xhtml"
          >IANA's reference list</a
        >.)
      </p>
    </div>
  </b-container>
</template>

<script>
import { humanDateTime } from "@/lib/date.js";
import tracking from "@/mixin/global_shared.js";
import auth from "@/mixin/auth.js";

export default {
  mixins: [tracking, auth],
  props: {
    results: {
      type: Object,
      default: null
    }
  },
  methods: {
    emailDate(d) {
      return humanDateTime(d);
    },
    statusClass: function(status) {
      return {
        sent: "delivery-status text-success",
        bounced: "delivery-status text-danger",
        deferred: "delivery-status text-warning",
        expired: "delivery-status text-danger"
      }[status];
    },
    hasOnlyOneDelivery: function(result) {
      return result.reduce((a, r) => a + r.number_of_attempts, 0) == 1;
    }
  }
};
</script>

<style lang="less">
.delivery-status + .delivery-status:before {
  content: " + ";
  color: #212529;
}
</style>
