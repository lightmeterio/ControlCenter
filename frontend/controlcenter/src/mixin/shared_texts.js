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
        "mailto:hello@lightmeter.io?subject=Feedback%20on%20Lightmeter&body=My%20thoughts%20on%20Lightmeter%3A%0A%0A%0AFirst%20installed%3A%20?%0AVersion%3A%20" +
        this.$appInfo.version +
        "%0AURL%3A%20" +
        window.location +
        "%0A"
    };
  }
};
