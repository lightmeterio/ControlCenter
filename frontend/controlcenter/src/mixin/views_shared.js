// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { getIsNotLoginOrNotRegistered } from "../lib/api.js";

export default {
  methods: {
    ValidSessionCheck: function() {
      let vue = this;
      let s = setInterval(function() {
        getIsNotLoginOrNotRegistered().catch(function() {
          vue.$router.push({ name: "login" });
        });
      }, 5000);
      return s;
    }
  }
};
