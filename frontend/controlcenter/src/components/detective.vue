<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <b-container class="mt-5">
    <b-form @submit.prevent="updateResults">
      <b-form-row class="justify-content-between align-items-end">
        <div class="col">
          <label>
            <!-- prettier-ignore -->
            <translate>Sender Email Address</translate>
          </label>
          <b-form-input
            type="email"
            name="mail_from"
            maxlength="255"
            required
            v-model="mail_from"
            :v-state="isEmailFrom"
            placeholder="sender@example.org"
          />
        </div>
        <div class="col">
          <label>
            <!-- prettier-ignore -->
            <translate>Recipient Email Address</translate>
          </label>
          <b-form-input
            type="email"
            name="mail_to"
            maxlength="255"
            required
            v-model="mail_to"
            :v-state="isEmailTo"
            placeholder="recipient@example.org"
          />
        </div>

        <div class="col">
          <label>
            <!-- prettier-ignore -->
            <translate>Time interval</translate>
          </label>
          <DateRangePicker
            @update="onUpdateDateRangePicker"
            :autoApply="autoApply"
            :opens="opens"
            :singleDatePicker="singleDatePicker"
            :alwaysShowCalendars="alwaysShowCalendars"
            :ranges="ranges"
            v-model="dateRange"
            :showCustomRangeCalendars="false"
            :max-date="new Date()"
          >
          </DateRangePicker>
        </div>

        <div class="col">
          <b-button type="submit" variant="primary">
            <!-- prettier-ignore -->
            <translate>Search</translate>
          </b-button>
        </div>
      </b-form-row>
    </b-form>

    <b-container ref="searchResultText" class="search-result-text mt-4">
      <p :class="searchResultClass">{{ searchResultText }}</p>
    </b-container>

    <detective-results :results="results.messages"></detective-results>

    <b-container class="pages mt-4 mb-4" v-show="results.last_page > 1">
      <button
        type="button"
        class="btn btn-outline-primary"
        v-for="p in results.last_page"
        :key="p"
        :disabled="p == results.page"
        @click="
          page = p;
          updateResults();
        "
      >
        {{ p }}
      </button>
    </b-container>

    <b-container v-show="forEndUsers" class="mt-5">
      <b-button variant="outline-primary" @click="escalateMessage">
        <!-- prettier-ignore -->
        <translate>Escalate</translate>
      </b-button>
    </b-container>
  </b-container>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import { humanDateTime } from "@/lib/date.js";
import { checkMessageDelivery, escalateMessage } from "@/lib/api.js";
import DateRangePicker from "@/3rd/components/DateRangePicker.vue";
import tracking from "@/mixin/global_shared.js";
import auth from "@/mixin/auth.js";
import datepicker from "@/mixin/datepicker.js";

function isEmail(email) {
  // NOTE: regexp also used in util/emailutil/email.go
  if (email == "") return null;
  return email.match(/^[^@\s]+@[^@\s]+$/) !== null;
}

export default {
  name: "detective",
  components: { DateRangePicker },
  mixins: [tracking, auth, datepicker],
  props: {
    forEndUsers: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      // detective-specific
      mail_from: "",
      mail_to: "",
      searchResultText: this.$gettext("No results yet"),
      searchResultClass: "text-muted",
      results: [],
      page: 1,

      // specific auth
      neededAuth: this.$route.name == "searchmessage" ? "detective" : "auth"

      // TODO: restrict timeInterval to 1 day if forEndUsers?
    };
  },
  computed: {
    isEmailFrom: function() {
      return isEmail(this.mail_from);
    },
    isEmailTo: function() {
      return isEmail(this.mail_to);
    }
  },
  methods: {
    updateSelectedInterval(obj) {
      let vue = this;
      vue.formatDatePickerValue(obj);
    },
    onUpdateDateRangePicker: function(obj) {
      this.trackEvent(
        "onUpdateDateRangePickerDetective",
        obj.startDate + "-" + obj.endDate
      );

      this.updateSelectedInterval(obj);
    },
    updateResults: function() {
      let vue = this;

      if (!this.isEmailFrom || !this.isEmailTo) {
        vue.searchResultClass = "text-warning";
        vue.searchResultText = vue.$gettext(
          "Please check the given email addresses"
        );
        return;
      }

      vue.searchResultClass = "text-muted";
      vue.searchResultText = "...";

      let interval = vue.buildDateInterval();

      checkMessageDelivery(
        this.mail_from,
        this.mail_to,
        interval.startDate,
        interval.endDate,
        vue.page
      ).then(function(response) {
        vue.results = response.data;

        let pageNb =
          vue.page > 1 ? " - " + vue.$gettext("Page") + " " + vue.page : "";

        vue.searchResultClass = vue.results.total
          ? "text-primary"
          : "text-secondary";
        vue.searchResultText = vue.results.total
          ? vue.results.total + " " + vue.$gettext("message(s) found") + pageNb
          : vue.$gettext("No message found");
        vue.$refs.searchResultText.scrollIntoView();
      });
    },
    escalateMessage() {
      let interval = this.buildDateInterval();
      escalateMessage(
        this.mail_from,
        this.mail_to,
        interval.startDate,
        interval.endDate
      ).then(function() {
        console.log("All good");
      }) 
    }
  },
  mounted() {
    this.updateSelectedInterval(this.dateRange);
  }
};
</script>

<style lang="less">
/* don't squeeze the inputs or datepicker too much, so they'll flex-wrap on smaller screens */
input,
.vue-daterange-picker {
  min-width: 200px;
}

.pages {
  display: flex;
  justify-content: center;

  button + button {
    margin-left: 0.5em;
  }
}
</style>
