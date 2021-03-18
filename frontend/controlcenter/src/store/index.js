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
    finishImportProgress(state) {
      state.isImportProgressFinished = true;
    }
  },
  actions: {
    setLanguageAction({ commit }, value) {
      commit("setLanguage", value);
    },
    setInsightsImportProgressFinished({ commit }) {
      commit("finishImportProgress");
    },
  },
  modules: {}
});
