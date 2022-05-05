// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

const DATE_YYYYMMDD = "YYYY-MM-DD";
const DATE_DMMM = "D MMM";

import moment from "moment";

function daysAgo(x) {
  let d = new Date();
  d.setHours(0, 0, 0, 0);
  d.setDate(d.getDate() - x);
  return d;
}

export default {
  data() {
    return {
      autoApply: true,
      alwaysShowCalendars: false,
      singleDatePicker: "range",
      dateRange: this.buildDefaultInterval(),
      ranges: this.defaultDatePickerRange(),
      opens: "center"
    };
  },
  methods: {
    formatDatePickerValue(obj) {
      let s = moment(obj.startDate).format(DATE_DMMM);
      let e = moment(obj.endDate).format(DATE_DMMM);

      document.querySelector(
        ".vue-daterange-picker .reportrange-text span"
      ).innerHTML = s == e ? s : s + " - " + e;
    },
    buildDefaultInterval() {
      // past month
      return {
        startDate: moment(daysAgo(29)).format(DATE_YYYYMMDD),
        endDate: moment(daysAgo(0)).format(DATE_YYYYMMDD)
      };
    },
    buildDateInterval() {
      let vue = this;

      let convert = function(date, offset) {
        let local = moment(moment(date).format(DATE_YYYYMMDD) + " " + offset);
        return local.utc().format(DATE_YYYYMMDD + " HH:mm:ss");
      };

      let start = convert(vue.dateRange.startDate, "00:00:00");
      let end = convert(vue.dateRange.endDate, "23:59:59");

      return { startDate: start, endDate: end };
    },
    defaultDatePickerRange() {
      let today = daysAgo(0);
      let yesterday = daysAgo(1);
      return {
        Today: [today, today],
        Yesterday: [yesterday, yesterday],
        "Last 7 days": [daysAgo(6), today],
        "Last 30 days": [daysAgo(29), today],
        "Last 3 months (all time)": [daysAgo(90), today]
      };
    }
  }
};
