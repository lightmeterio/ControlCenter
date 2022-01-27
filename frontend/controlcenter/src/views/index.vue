<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div id="insights-page" class="d-flex flex-column min-vh-100">
    <mainheader></mainheader>
    <div class="container-fluid greet-panel">
      <div class="container main-content">
        <div class="row align-items-center">
          <div class="col-auto">
            <h1 class="row-title">
              <!-- prettier-ignore -->
              <translate>Observatory</translate>
            </h1>
            <div class="panel panel-default greeting">
              <h3>{{ welcomeUserText }}, {{ greetingText }}</h3>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="container-fluid grey">
      <div class="container">
        <tabs class :tabList="tabList">
          <template v-slot:tabPanel-1>
            <div class="container">
              <div
                class="row d-flex align-items-center justify-content-between time-interval card-section-heading"
              >
                <div class="col-lg-6 col-md-6 col-12 p-0">
                  <form id="insights-form">
                    <div
                      class="form-group d-flex justify-content-start align-items-center"
                    >
                      <label class="sr-only">
                        <!-- prettier-ignore -->
                        <translate>Filter</translate>
                      </label>
                      <select
                        id="insights-filter"
                        class="form-control custom-select custom-select-sm"
                        name="filter"
                        form="insights-form"
                        v-model="insightsFilter"
                        v-on:change="updateInsights"
                      >
                        <!-- todo remove in style -->
                        <option
                          selected
                          v-on:click="
                            trackClick(
                              'InsightsFilterCategoryHomepage',
                              'Active'
                            )
                          "
                          value="category-active"
                        >
                          <!-- prettier-ignore -->
                          <translate>Active</translate>
                        </option>
                        <option value="nofilter">
                          <!-- prettier-ignore -->
                          <translate>All</translate>
                        </option>
                        <!--    " -->
                        <option
                          v-on:click="
                            trackClick(
                              'InsightsFilterCategoryHomepage',
                              'Local'
                            )
                          "
                          value="category-local"
                        >
                          <!-- prettier-ignore -->
                          <translate>Local</translate>
                        </option>
                        <option
                          v-on:click="
                            trackClick('InsightsFilterCategoryHomepage', 'News')
                          "
                          value="category-news"
                        >
                          <!-- prettier-ignore -->
                          <translate>News</translate>
                        </option>
                        <option
                          v-on:click="
                            trackClick(
                              'InsightsFilterCategoryHomepage',
                              'Intel'
                            )
                          "
                          value="category-intel"
                        >
                          <!-- prettier-ignore -->
                          <translate>Intel</translate>
                        </option>
                      </select>
                      <select
                        id="insights-sort"
                        class="form-control custom-select custom-select-sm"
                        name="order"
                        form="insights-form"
                        v-model="insightsSort"
                        v-on:change="updateInsights"
                      >
                        <!-- todo remove in style -->
                        <option
                          v-on:click="
                            trackClick('InsightsFilterOrderHomepage', 'Newest')
                          "
                          selected
                          value="creationDesc"
                        >
                          <!-- prettier-ignore -->
                          <translate>Newest</translate>
                        </option>
                        <option
                          v-on:click="
                            trackClick('InsightsFilterOrderHomepage', 'Oldest')
                          "
                          value="creationAsc"
                        >
                          <!-- prettier-ignore -->
                          <translate>Oldest</translate>
                        </option>
                      </select>
                    </div>
                  </form>
                </div>

                <div class="d-flex align-items-center">
                  <input
                    type="checkbox"
                    v-on:click="
                      trackClick('InsightsFilterCategoryHomepage', 'Archived')
                    "
                    value="category-archived"
                    name="Archived"
                  />
                  <!-- prettier-ignore -->
                  <label class="mb-0 ml-1" for="Archived">
                    <translate>Show Archived</translate>
                  </label>
                </div>

                <div
                  class="col-lg-4 col-md-4 col-12 p-0 justify-content-start justify-content-lg-end d-flex"
                >
                  <label class="col-md-2 col-form-label sr-only">
                    <!-- prettier-ignore -->
                    <translate>Time interval</translate>:
                  </label>
                  <div class="p-1">
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
                    ></DateRangePicker>
                  </div>
                  <div class="p-1">
                    <b-button
                      variant="primary"
                      size="sm"
                      @click="downloadRawLogsInInterval"
                      :disabled="rawLogsDownloadsDisable"
                    >
                      <i
                        class="fas fa-download"
                        style="margin-right: 0.25rem;"
                      ></i>
                      <translate>Logs</translate>
                    </b-button>
                  </div>
                </div>
              </div>
            </div>
            <insights
              class="row"
              v-show="shouldShowInsights"
              :insights="insights"
              @dateIntervalChanged="handleExternalDateIntervalChanged"
            ></insights>

            <import-progress-indicator
              :label="generatingInsights"
              @finished="handleProgressFinished"
            ></import-progress-indicator>
          </template>

          <template v-slot:tabPanel-2>
            <graphdashboard
              :graphDateRange="dashboardInterval"
            ></graphdashboard>
            <b-toaster
              ref="statusMessage"
              name="statusMessage"
              class="status-message"
            ></b-toaster>
          </template>
        </tabs>
      </div>
    </div>

    <mainfooter></mainfooter>
  </div>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import {
  linkToRawLogsInInterval,
  countLogLinesInInterval,
  fetchInsights,
  getIsNotLoginOrNotRegistered,
  getUserInfo,
  getStatusMessage
} from "../lib/api.js";

import tracking from "../mixin/global_shared.js";
import shared_texts from "../mixin/shared_texts.js";
import auth from "../mixin/auth.js";
import datepicker from "@/mixin/datepicker.js";
import { mapActions, mapState } from "vuex";
import DateRangePicker from "vue2-daterange-picker";
import "vue2-daterange-picker/dist/vue2-daterange-picker.css";
import tabs from "../components/tabs";

export default {
  name: "insight",
  components: { DateRangePicker, tabs },
  mixins: [tracking, shared_texts, auth, datepicker],
  data() {
    return {
      username: "",
      updateDashboardAndInsightsIntervalID: null,
      dashboardInterval: this.buildDefaultInterval(),
      insightsFilter: "category-active",
      insightsSort: "creationDesc",
      insights: [],
      tabList: ["Insights", "Graphs"],

      // log import progress
      generatingInsights: this.$gettext("Generating insights"),

      rawLogsDownloadsDisable: true,

      statusMessage: null,
      statusMessageId: null
    };
  },
  created() {},
  computed: {
    shouldShowInsights() {
      return this.isImportProgressFinished;
    },
    greetingText() {
      // todo use better translate function for weekdays
      let dateObj = new Date();
      let weekday = dateObj.toLocaleString("default", { weekday: "long" });
      let translation = this.$gettext(
        "you have %{weekday} new Local and %{weekday} new Intel insights since your last visit."
      );
      let message = this.$gettextInterpolate(translation, { weekday: weekday });
      return message;
    },
    welcomeUserText() {
      let translation = this.$gettext("Hi %{username}");
      let message = this.$gettextInterpolate(translation, {
        username: this.username
      });
      return message;
    },
    ...mapState(["isImportProgressFinished"])
  },
  methods: {
    updateSelectedInterval(obj) {
      let vue = this;
      vue.updateDashboardAndInsights();
      vue.formatDatePickerValue(obj);
      vue.updateRawLogsDownloadButton();
    },
    handleProgressFinished() {
      this.setInsightsImportProgressFinished();
      this.updateDashboardAndInsights();
    },
    updateDashboardAndInsights() {
      let vue = this;
      vue.updateInsights();
      vue.updateDashboard();
      vue.updateStatusMessage();
    },
    handleExternalDateIntervalChanged(obj) {
      if (obj === undefined) {
        this.updateSelectedInterval(this.dateRange);
        return;
      }

      this.dateRange = obj;
      this.insightsFilter = "category-" + obj.category;
      this.updateSelectedInterval(obj);
    },
    onUpdateDateRangePicker: function(obj) {
      this.trackEvent(
        "onUpdateDateRangePicker",
        obj.startDate + "-" + obj.endDate
      );

      this.updateSelectedInterval(obj);
    },
    updateRawLogsDownloadButton: function() {
      let vue = this;
      let interval = vue.buildDateInterval();

      countLogLinesInInterval(interval.startDate, interval.endDate).then(
        function(response) {
          vue.rawLogsDownloadsDisable = response.data.count == 0;
        }
      );
    },
    downloadRawLogsInInterval() {
      let interval = this.buildDateInterval();
      let link = linkToRawLogsInInterval(interval.startDate, interval.endDate);
      let range = interval.startDate + "_" + interval.endDate;

      this.trackEvent("DownloadDatePickerLogs", range);

      window.open(link);
    },
    onStatusMessageClosed() {
      this.trackEvent("CloseStatusMessage", this.statusMessageId);
    },
    updateStatusMessage: function() {
      let vue = this;

      getStatusMessage().then(function(response) {
        let notification =
          response.data !== null ? response.data.notification : null;

        if (notification === null || notification.title == "") {
          return;
        }

        let id = response.data.id;

        let isNew =
          vue.statusMessage === null ||
          vue.statusMessage.message != notification.message ||
          vue.statusMessage.title != notification.title;

        vue.statusMessage = notification;
        vue.statusMessageId = id;

        if (!isNew) {
          return;
        }

        const e = vue.$createElement;

        const msg = [
          e("p", vue.statusMessage.message),
          e(
            "a",
            { attrs: { href: vue.statusMessage.action.link } },
            vue.statusMessage.action.label
          )
        ];

        vue.$bvToast.toast([msg], {
          variant: vue.statusMessage.severity,
          title: vue.statusMessage.title,
          noAutoHide: true,
          toaster: "statusMessage",
          solid: true
        });
      });
    },
    updateInsights: function() {
      let vue = this;
      let interval = vue.buildDateInterval();

      fetchInsights(
        interval.startDate,
        interval.endDate,
        vue.insightsFilter,
        vue.insightsSort
      ).then(function(response) {
        vue.insights = response.data;
      });
    },
    updateDashboard: function() {
      let vue = this;
      let interval = vue.buildDateInterval();
      vue.dashboardInterval = interval;
    },
    initIndex: function() {
      let vue = this;

      vue.updateSelectedInterval(vue.dateRange);

      this.updateDashboardAndInsightsIntervalID = window.setInterval(
        function() {
          getIsNotLoginOrNotRegistered().then(vue.updateDashboardAndInsights);
        },
        30000
      );
    },
    ...mapActions(["setInsightsImportProgressFinished"])
  },
  mounted() {
    this.initIndex();

    let vue = this;

    vue.$root.$on("bv::toast:hidden", event => {
      vue.onStatusMessageClosed(event);
    });

    getUserInfo().then(function(response) {
      vue.username = response.data.user.name;
    });
  },
  destroyed() {
    window.clearInterval(this.updateDashboardAndInsightsIntervalID);
  }
};
</script>

<style lang="less">
#insights-page .greeting h3 {
  font-size: 18px;
  font-weight: 500;
  margin: 0;
  text-align: left;
  color: #111827;
}

#insights-page .container-fluid.greet-panel {
  background: #e2f5fc;
}

#insights-page .container-fluid.grey {
  background: #f9fafb;
}

#insights-page .card-section-heading {
  background-color: #f9f9f9;
}

#insights-page .card-section-heading h2 {
  font-size: 24px;
  font-weight: bold;
  margin: 0;
}

#insights-page .time-interval .form-group {
  margin: 0;
  padding: 0;
}

#insights-page .form-control.custom-select {
  margin: 0 0.5em 0 0;
  padding: 10px 16px;
  background-color: #fff;
  border: 1px solid #d1d5db;
  box-shadow: 0px 1px 2px rgba(0, 0, 0, 0.05);
  border-radius: 6px;
  color: #374151;
  font-weight: 500;
  font-size: 14px;
  height: 100%;
}

#insights-page .form-control.custom-select option {
  background-color: #fff !important;
  box-shadow: 0px 1px 2px rgba(0, 0, 0, 0.05);
  border-radius: 6px;
}

#insights-page .daterangepicker .ranges ul {
  margin: 0 auto auto auto;
}

#insights-page .greeting {
  background-color: #b6e6f6;
  padding: 10px 16px;
  border-radius: 8px;
  margin-bottom: 30px;
}

#insights-page h1.row-title {
  font-size: 32px;
  font-weight: bold;
  margin: 0.7em 0 0.4em 0;
  text-align: left;
}

#insights-page .row-title span {
  color: #185a8d;
}

#insights-page .vue-daterange-picker .reportrange-text {
  background: #daebf4;
  cursor: pointer;
  padding: 0.3rem 1rem;
  border: none;
  font-size: 12px;
  color: #00689d;
  font-weight: bold;
  border-radius: 5px;
  text-align: center;
  cursor: pointer;
  display: flex;
  justify-content: center;

  span {
    order: 1;
    margin-top: 0.25em;
  }
  svg {
    order: 2;
    margin-left: 1em;
    margin-top: 0.45em;
  }
}

#insights-page .insights-title {
  text-align: left;
}

@media (max-width: 768px) {
  #insights-page .daterangepicker.dropdown-menu {
    left: -40vw;
  }
  #insights-page .vue-daterange-picker .calendars {
    flex-wrap: wrap;
  }
  .daterangepicker .calendars-container {
    display: block;
  }
  #insights-page .vue-daterange-picker {
    padding: 0px;
  }
  #insights-page .vue-daterange-picker .form-control {
    max-width: inherit;
  }

  #insights-page #insights {
    min-height: 100vh;
  }
}

@media (min-width: 768px) and (max-width: 1024px) {
  #insights-page #insights {
    min-height: 60vh;
  }
}

.progress-indicator-area {
  margin-top: 60px;
  margin-bottom: 60px;
}

.b-toaster.status-message {
  max-width: 100%;
  width: 100%;

  .b-toast,
  .toast {
    max-width: 100%;
    width: 100%;
    flex-basis: 100%;
    margin-top: 1rem;
    margin-bottom: 1.9rem;
  }
  .toast-body {
    text-align: left;
    > * {
      margin: 1em;
      display: block;
    }
  }
}
</style>
