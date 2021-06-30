// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { getApplicationInfo } from "@/lib/api.js";

export default {
  data() {
    return {
      PublicIPHelpText: this.$gettext(
        "Lightmeter will check your IP against the most popular available RBLs and detect blocks"
      ),
      FeedbackButtonTitle: this.$gettext("What would you improve?"),
      AppVersion: null
    };
  },
  mounted() {
    let vue = this;
    getApplicationInfo().then(function(response) {
      vue.AppVersion = response.data.version;
    });
  },
  computed: {
    FeedbackMailtoLink() {
      let vue = this;

      return (
        "mailto:hello@lightmeter.io?subject=" +
        encodeURIComponent(this.$gettext("Feedback on Lightmeter")) +
        "&body=" +
        encodeURIComponent(
          this.$gettext("My thoughts on Lightmeter") +
            ":\n\n\n" +
            this.$gettext("First installed") +
            ": ?\n" +
            this.$gettext("Version") +
            ": " +
            vue.AppVersion +
            "\nURL: " +
            window.location +
            "\n"
        )
      );
    }
  }
};
