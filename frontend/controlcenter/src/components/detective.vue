<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <b-container>
    <b-form>
      <b-form-row class="justify-content-around">
        <div class="col-auto">
          <label class="sr-only">
            <!-- prettier-ignore -->
            <translate>Sender Email Address</translate>:
          </label>
          <b-form-input
            type="email"
            name="mail_from"
            maxlength="255"
            required
            v-model="mail_from"
            :v-state="isEmailFrom"
            placeholder="Sender Email Address"
            @focusout="updateResults"
            @keyup.enter="updateResults" />
        </div>
        <div class="col-auto">
          <label class="sr-only">
            <!-- prettier-ignore -->
            <translate>Recipient Email Address</translate>:
          </label>
          <b-form-input
            type="email"
            name="mail_to"
            maxlength="255"
            required
            v-model="mail_to"
            :v-state="isEmailTo"
            placeholder="Recipient Email Address"
            @focusout="updateResults"
            @keyup.enter="updateResults" />
        </div>

        <div class="col-auto">
          <label class="sr-only">
            <!-- prettier-ignore -->
            <translate>Time interval</translate>:
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
      </b-form-row>
    </b-form>
    
    <b-container class="search-result-text">
      <p :class="searchResultClass">{{ searchResultText }}</p>
    </b-container>
    
    <import-progress-indicator ref="import" :label="importingLogs" @finished="handleProgressFinished"></import-progress-indicator>
    
    <b-container class="results">
      <b-row v-for="result in results" :key="result.Timestamp">
        <b-col sm>{{ emailDate(result.time) }}</b-col>
        <b-col sm>Status: {{ result.status }}</b-col>
        <b-col sm>DSN: {{ result.dsn }}</b-col>
        <!-- TODO: use list of DSNs https://www.iana.org/assignments/smtp-enhanced-status-codes/smtp-enhanced-status-codes.xhtml -->
      </b-row>
    </b-container>
    
    <b-container v-show="forEndUsers">
      <input type="submit" value="Escalate" />
    </b-container>
    
  </b-container>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import { humanDateTime } from "@/lib/date.js";
import { checkMessageDelivery } from "@/lib/api.js";

import DateRangePicker from "@/3rd/components/DateRangePicker.vue";
import tracking from "@/mixin/global_shared.js";
import auth from "@/mixin/auth.js";
import datepicker from "@/mixin/datepicker.js";
import { mapActions } from "vuex";


function isEmail (email) {
  // NOTE: regexp also used in util/emailutil/email.go
  if (email == '')
    return null;
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
    },
  },
  data() {
    return {
      // detective-specific
      mail_from: "",
      mail_to: "",
      searchResultText: '',
      searchResultClass: '',
      results: [],
      
      // logs import
      importingLogs: this.$gettext("Importing logs"),
      
      // TODO: restrict timeInterval to 1 day if forEndUsers?
    };
  },
  computed: {
    isEmailFrom: function () { return isEmail(this.mail_from); },
    isEmailTo:   function () { return isEmail(this.mail_to  ); },
  },
  methods: {
    updateSelectedInterval(obj) {
      let vue = this;
      vue.updateResults();
      vue.formatDatePickerValue(obj);
    },
    emailDate(d) {
      return humanDateTime(d);
    },
    handleProgressFinished() {
      this.setInsightsImportProgressFinished();
      this.updateResults();
    },
    onUpdateDateRangePicker: function(obj) {
      this.trackEvent(
        "onUpdateDateRangePicker",
        obj.startDate + "-" + obj.endDate
      );

      this.updateSelectedInterval(obj);
    },
    updateResults: function () {
      let vue = this;
      
      if (!this.isEmailFrom || !this.isEmailTo) {
        vue.searchResultClass = 'text-warning';
        vue.searchResultText = vue.$gettext('Please check the given email addresses');
        return;
      }
      
      vue.searchResultClass = 'text-muted';
      vue.searchResultText = '...';
      
      let interval = vue.buildDateInterval();
      
      checkMessageDelivery(this.mail_from, this.mail_to, interval.startDate, interval.endDate).then( function (response) {
        vue.results = response.data;
        vue.searchResultClass = vue.results.length ? 'text-primary' : 'text-secondary';
        vue.searchResultText = vue.results.length ? vue.results.length + ' ' + vue.$gettext('messages found') : vue.$gettext('No message found');
      });
    },
    ...mapActions(["setInsightsImportProgressFinished"])
  }
};
</script>

<style lang="less">
</style>
