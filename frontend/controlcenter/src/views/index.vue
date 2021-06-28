<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div id="insights-page" class="d-flex flex-column min-vh-100">
    <mainheader></mainheader>
    <div class="container main-content">
      <div class="row align-items-center">
        <div class="col-7 col-sm-8 col-md-9 col-lg-10">
          <h1 class="row-title">
            <!-- prettier-ignore -->
            <translate>Observatory</translate>
          </h1>
        </div>
        <div class="col-5 col-sm-4 col-md-3 col-lg-2">
          <a
            :href="FeedbackMailtoLink"
            :title="FeedbackButtonTitle"
            class="btn btn-sm btn-block btn-outline-info"
            ><i
              class="fas fa-star"
              style="margin-right: 0.25rem;"
              data-toggle="tooltip"
              data-placement="bottom"
            ></i>
            <!-- prettier-ignore -->
            <translate>Feedback</translate>
          </a>
        </div>
      </div>

      <div class="row">
        <div class="col-md-12">
          <div class="panel panel-default greeting">
            <div class="row">
              <div class="col-md-3 align-center">
                <img
                  class="hero"
                  src="@/assets/greeting-observatory.svg"
                  alt="Observatory illustration"
                />
              </div>

              <div class="col-md-9 d-flex align-items-center">
                <div class="row">
                  <div class="container">
                    <h3>{{ greetingText }}</h3>
                    <p>{{ welcomeUserText }}</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <graphdashboard :graphDateRange="dashboardInterval"></graphdashboard>

      <div
        class="row container d-flex align-items-center time-interval card-section-heading"
      >
        <div class="col-lg-2 col-md-6 col-6 p-2">
          <h2 class="insights-title">
            <!-- prettier-ignore -->
            <translate>Insights</translate>
          </h2>
        </div>
        <div class="col-lg-3 col-md-6 col-6 p-2">
          <label class="col-md-2 col-form-label sr-only">
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

        <div class="col-lg-4 col-md-6 col-12 ml-auto p-2">
          <form id="insights-form">
            <div
              class="form-group d-flex justify-content-end align-items-center"
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
                style="width: 33%"
                v-on:change="updateInsights"
              >
                <!-- todo remove in style -->
                <option
                  selected
                  v-on:click="
                    trackClick('InsightsFilterCategoryHomepage', 'Active')
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
                    trackClick('InsightsFilterCategoryHomepage', 'Local')
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
                    trackClick('InsightsFilterCategoryHomepage', 'Archived')
                  "
                  value="category-archived"
                >
                  <!-- prettier-ignore -->
                  <translate>Archived</translate>
                </option>
              </select>
              <select
                id="insights-sort"
                class="form-control custom-select custom-select-sm"
                name="order"
                form="insights-form"
                v-model="insightsSort"
                style="width: 38%"
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
      </div>

      <import-progress-indicator
        :label="generatingInsights"
        @finished="handleProgressFinished"
      ></import-progress-indicator>

      <insights
        class="row"
        v-show="shouldShowInsights"
        :insights="insights"
        @dateIntervalChanged="handleExternalDateIntervalChanged"
      ></insights>
    </div>
    <mainfooter></mainfooter>
  </div>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import {
  fetchInsights,
  getIsNotLoginOrNotRegistered,
  getUserInfo
} from "../lib/api.js";

import DateRangePicker from "../3rd/components/DateRangePicker.vue";
import tracking from "../mixin/global_shared.js";
import shared_texts from "../mixin/shared_texts.js";
import auth from "../mixin/auth.js";
import datepicker from "@/mixin/datepicker.js";
import { mapActions, mapState } from "vuex";

export default {
  name: "insight",
  components: { DateRangePicker },
  mixins: [tracking, shared_texts, auth, datepicker],
  data() {
    return {
      username: "",
      updateDashboardAndInsightsIntervalID: null,
      dashboardInterval: this.buildDefaultInterval(),
      insightsFilter: "nofilter",
      insightsSort: "creationDesc",
      insights: [],

      // log import progress
      generatingInsights: this.$gettext("Generating insights")
    };
  },
  computed: {
    shouldShowInsights() {
      return this.isImportProgressFinished;
    },
    greetingText() {
      // todo use better translate function for weekdays
      let dateObj = new Date();
      let weekday = dateObj.toLocaleString("default", { weekday: "long" });
      let translation = this.$gettext("Happy %{weekday}");
      let message = this.$gettextInterpolate(translation, { weekday: weekday });
      return message;
    },

    welcomeUserText() {
      let translation = this.$gettext("and welcome back, %{username}");
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
    },
    handleProgressFinished() {
      this.setInsightsImportProgressFinished();
      this.updateDashboardAndInsights();
    },
    updateDashboardAndInsights() {
      let vue = this;
      vue.updateInsights();
      vue.updateDashboard();
    },
    handleExternalDateIntervalChanged(obj) {
      this.dateRange = obj;
      if (obj.category !== undefined) {
        this.insightsFilter = "category-" + obj.category;
      }
      this.updateSelectedInterval(obj);
    },
    onUpdateDateRangePicker: function(obj) {
      this.trackEvent(
        "onUpdateDateRangePicker",
        obj.startDate + "-" + obj.endDate
      );

      this.updateSelectedInterval(obj);
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

    getUserInfo().then(function(response) {
      vue.username = response.data.Name;
    });
  },
  destroyed() {
    window.clearInterval(this.updateDashboardAndInsightsIntervalID);
  }
};
</script>

<style lang="less">
#insights-page .greeting h3 {
  font: 22px/32px Inter;
  font-weight: bold;
  margin: 0;
  text-align: left;
  color: white;
}

#insights-page .greeting p {
  text-align: left;
}

#insights-page .card-section-heading {
  background-color: #f9f9f9;
}

#insights-page .time-interval {
  margin: 0.6rem 0 0 0;
  border-radius: 10px;
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

#insights-page #insights-form select {
  font-size: 12px;
  border-radius: 5px;
  margin-right: 0.2rem;
}

#insights-page .form-control.custom-select {
  margin: 0;
  background-color: #e6e7e7;
  color: #202324;
}

#insights-page .greeting {
  background: url(~@/assets/greeting-lensflare.svg) no-repeat right top,
    linear-gradient(104deg, #2a93d6 0%, #3dd9d6 100%) 0% 0% padding-box;
  color: white;
  padding: 0.5rem;
  border-radius: 7px;
  margin-bottom: 30px;
}

#insights-page h1.row-title {
  font-size: 32px;
  font-weight: bold;
  margin: 0.7em 0 0.8em 0;
  text-align: left;
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
}

#insights-page .vue-daterange-picker .reportrange-text {
  display: flex;
  justify-content: center;
}

#insights-page .vue-daterange-picker .reportrange-text span {
  order: 1;
  margin-top: 0.25em;
}

#insights-page .vue-daterange-picker .reportrange-text svg {
  order: 2;
  margin-left: 1em;
  margin-top: 0.45em;
}

#insights-page #insights {
  min-height: 30vh;
}

#insights-page .modebar {
  display: none;
}

#insights-page .vue-daterange-picker .calendars {
  flex-wrap: nowrap;
}

#insights-page .insights-title {
  text-align: left;
}

@media (min-width: 768px) {
  #insights-page .greeting {
    height: 150px;
  }
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
    max-width: 150px;
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
  #insights-page .vue-daterange-picker {
    max-width: none;
  }
  #insights-page .daterangepicker.dropdown-menu {
    left: -10vw;
  }
  #insights-page #insights {
    min-height: 60vh;
  }
  #insights-page .vue-daterange-picker .calendars {
    flex-wrap: wrap;
  }
  #insights-page .daterangepicker .calendars .ranges li:last-child {
    display: block;
  }
}

.progress-indicator-area {
  margin-top: 60px;
  margin-bottom: 60px;
}
</style>
