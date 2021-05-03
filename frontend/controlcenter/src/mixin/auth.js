// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import {
  getIsNotLoginOrNotRegistered,
  getIsNotLoginAndNotEndUsersEnabled
} from "../lib/api.js";

export default {
  data() {
    return {
      neededAuth: "auth",
      sessionInterval: null
    };
  },
  methods: {
    ValidSessionCheck: function() {
      let vue = this;
      let s = setInterval(function() {
        switch (vue.neededAuth) {
          case "detective":
            getIsNotLoginAndNotEndUsersEnabled().catch(function() {
              window.location.reload(); // refresh page, index.js will do the rest
            });
            break;
          default:
            getIsNotLoginOrNotRegistered().catch(function() {
              vue.$router.push({ name: "login" });
            });
        }
      }, 5000);
      return s;
    }
  },
  mounted() {
    this.sessionInterval = this.ValidSessionCheck();
  },
  destroyed() {
    window.clearInterval(this.sessionInterval);
  }
};
