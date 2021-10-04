// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import Vue from "vue";
import Vuex from "vuex";

import { requestWalkthroughCompletedStatus } from "../lib/api.js";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    language: "",
    isImportProgressFinished: false,
    walkthroughNeedsToRun: false,
    alerts: []
  },
  mutations: {
    setLanguage(state, language) {
      state.language = language;
    },
    finishImportProgress(state, wait) {
      setTimeout(function() {
        state.isImportProgressFinished = true;
      }, wait * 1000);
    },
    setWalkthroughState(state, value) {
      state.walkthroughNeedsToRun = value;
      if (!value)
        // only ever set the walkthrough completeness to true, never reset it to false
        requestWalkthroughCompletedStatus(true);
    },
    newAlert(state, alert) {
      let vue = this;
      let id = Date.now();
      alert.id = id;
      state.alerts.push(alert);
      setTimeout(function() {
        vue.commit("removeAlert", id);
      }, 5000);
    },
    removeAlert(state, id) {
      state.alerts = state.alerts.filter(a => a.id != id);
    }
  },
  actions: {
    setLanguageAction({ commit }, value) {
      commit("setLanguage", value);
    },
    setInsightsImportProgressFinished({ commit }, options) {
      options = Object.assign({ wait: 0 }, options);
      commit("finishImportProgress", options.wait);
    },
    setWalkthroughNeedsToRunAction({ commit }, value) {
      commit("setWalkthroughState", value);
    },
    newAlertError({ commit }, message) {
      commit("newAlert", { severity: "danger", message: message });
    },
    newAlertSuccess({ commit }, message) {
      commit("newAlert", { severity: "success", message: message });
    }
  },
  modules: {}
});
