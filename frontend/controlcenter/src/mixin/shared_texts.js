// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

export default {
  data() {
    return {
      PublicIPHelpText: this.$gettext(
        "Lightmeter will check your IP against the most popular available RBLs and detect blocks"
      )
    };
  }
};
