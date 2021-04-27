// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only


const DATE_YYYYMMDD = "YYYY-MM-DD";
const DATE_DMMM = "D MMM";

import moment from "moment";


export default {
  data() {
    return {
      autoApply: true,
      alwaysShowCalendars: false,
      singleDatePicker: false,
      dateRange: this.buildDefaultInterval(),
      ranges: this.defaultDatePickerRange(),
      opens: "right"
    };
  },
  methods: {
    formatDatePickerValue(obj) {
      document.querySelector(
        ".vue-daterange-picker .reportrange-text span"
      ).innerHTML =
        moment(obj.startDate).format(DATE_DMMM) +
        " - " +
        moment(obj.endDate).format(DATE_DMMM);
    },
    buildDefaultInterval() {
      // past month
      return {
        startDate: moment().subtract(29, "days").format(DATE_YYYYMMDD),
        endDate: moment().format(DATE_YYYYMMDD),
      };
    },
    buildDateInterval() {
      let vue = this;
      let start = moment(vue.dateRange.startDate).format(DATE_YYYYMMDD);
      let end = moment(vue.dateRange.endDate).format(DATE_YYYYMMDD);

      return {startDate: start, endDate: end};
    },
    defaultDatePickerRange() {
      let today = new Date();
      today.setHours(0, 0, 0, 0);
      let yesterday = new Date();
      yesterday.setDate(today.getDate() - 1);
      yesterday.setHours(0, 0, 0, 0);
      let thisMonthStart = new Date(today.getFullYear(), today.getMonth(), 1);
      let thisMonthEnd = new Date(today.getFullYear(), today.getMonth() + 1, 0);
      let lastMonthStart = new Date(today.getFullYear(), today.getMonth() - 1, 1);
      let lastMonthEnd = new Date(today.getFullYear(), today.getMonth() - 1 + 1, 0);
      return {
        Today: [today, today],
        Yesterday: [yesterday, yesterday],
        "This month": [thisMonthStart, thisMonthEnd],
        "Last month": [lastMonthStart, lastMonthEnd],
        "This year": [
          new Date(today.getFullYear(), 0, 1),
          new Date(today.getFullYear(), 11, 31)
        ]
      };
    }
  }
};

