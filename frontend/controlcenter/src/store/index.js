// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import Vue from "vue";
import Vuex from "vuex";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    language: ""
  },
  mutations: {
    setLanguage(state, language) {
      state.language = language;
    }
  },
  actions: {
    setLanguageAction({ commit }, value) {
      commit("setLanguage", value);
    }
  },
  modules: {}
});
