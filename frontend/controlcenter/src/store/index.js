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
    walkthroughNeedsToRun: false
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
    }
  },
  modules: {}
});
