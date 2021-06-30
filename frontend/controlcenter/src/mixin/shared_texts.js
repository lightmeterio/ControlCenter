// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

export default {
  data() {
    return {
      PublicIPHelpText: this.$gettext(
        "Lightmeter will check your IP against the most popular available RBLs and detect blocks"
      ),
      FeedbackButtonTitle: this.$gettext("What would you improve?"),
      FeedbackMailtoLink:
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
            this.$appInfo.version +
            "\nURL: " +
            window.location +
            "\n"
        )
    };
  }
};
