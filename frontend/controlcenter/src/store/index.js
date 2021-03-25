// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import Vue from "vue";
import Vuex from "vuex";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    language: "",
    isImportProgressFinished: false
  },
  mutations: {
    setLanguage(state, language) {
      state.language = language;
    },
    finishImportProgress(state, wait) {
      setTimeout( function () { state.isImportProgressFinished = true;}, wait*1000);
    }
  },
  actions: {
    setLanguageAction({ commit }, value) {
      commit("setLanguage", value);
    },
    setInsightsImportProgressFinished({ commit }, options) {
      options = Object.assign({wait: 0}, options);
      commit("finishImportProgress", options.wait);
    },
  },
  modules: {}
});
