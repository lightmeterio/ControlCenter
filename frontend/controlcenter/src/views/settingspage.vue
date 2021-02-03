<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<template>
  <div class="settings-page d-flex flex-column min-vh-100">
    <mainheader></mainheader>

    <b-container id="settings" class="main-content">
      <h2 class="form-heading">
        <!-- prettier-ignore -->
        <translate>Settings</translate>
      </h2>
      <div class="form-container">
        <h6 class="form-heading">
          <!-- prettier-ignore -->
          <translate>Notifications</translate>
        </h6>
        <b-form
          @submit="onNotificationSettingsSubmit"
          id="notifications-form-container"
        >
          <b-form-group :label="SlackNotifications" class="slack-disabler">
            <b-form-radio-group
              class="pt-2"
              required
              v-model="settings.slack_notifications.enabled"
              :options="SlackNotificationsSwitchOptions"
            ></b-form-radio-group>
          </b-form-group>

          <b-form-group label="Slack message language" class="slack-language">
            <b-form-radio-group
              class="pt-2"
              required
              v-model="settings.slack_notifications.language"
              :options="languages"
              stacked
            ></b-form-radio-group>
          </b-form-group>

          <b-form-group
            class="slack-channel"
            :label="SlackChannel"
            label-for="slackChannel"
          >
            <b-form-input
              name="messenger_channel"
              id="slackChannel"
              v-model="settings.slack_notifications.channel"
              required
              :placeholder="SlackChannelInputPlaceholder"
              maxlength="255"
            ></b-form-input>
          </b-form-group>

          <b-form-group
            class="slack-token"
            :label="SlackAPItoken"
            label-for="slackApiToken"
          >
            <b-form-input
              name="messenger_token"
              id="slackApiToken"
              v-model="settings.slack_notifications.bearer_token"
              required
              :placeholder="SlackAPItokenPlacefolder"
              maxlength="255"
            ></b-form-input>
          </b-form-group>

          <div class="button-group">
            <b-button variant="primary" class="general-save" type="submit">
              <!-- prettier-ignore -->
              <translate>Save</translate>
            </b-button>
            <b-button
              variant="primary"
              class="general-cancel btn-cancel"
              type="submit"
            >
              <!-- prettier-ignore -->
              <translate>Cancel</translate>
            </b-button>
          </div>
        </b-form>

        <h6 class="form-heading">
          <!-- prettier-ignore -->
          <translate>General</translate>
        </h6>

        <b-form @submit="onGeneralSettingsSubmit" id="general-form-container">
          <b-form-group
            class="postfixPublicIP"
            :label="PostfixPublicIP"
            label-for="postfixPublicIP"
          >
            <b-form-input
              name="postfixPublicIP"
              id="postfixPublicIP"
              v-model="settings.general.postfix_public_ip"
              required
              :placeholder="EnterIpAddress"
              maxlength="255"
            ></b-form-input>
          </b-form-group>

          <div class="button-group">
            <b-button variant="primary" class="general-save" type="submit">
              <!-- prettier-ignore -->
              <translate>Save</translate>
            </b-button>
            <b-button
              variant="primary"
              class="general-cancel btn-cancel"
              type="submit"
            >
              <!-- prettier-ignore -->
              <translate>Cancel</translate>
            </b-button>
          </div>
        </b-form>
      </div>
    </b-container>
    <mainfooter></mainfooter>
  </div>
</template>

<script>
import { getSettings } from "../lib/api.js";
import { getMetaLanguage } from "../lib/api.js";
import { submitNotificationsSettingsForm } from "../lib/api.js";
import { submitGeneralForm } from "../lib/api.js";
import session from "@/mixin/views_shared";

export default {
  name: "settingspage",
  components: {},
  mixins: [session],
  data() {
    return {
      sessionInterval: null,
      settings: {
        slack_notifications: {
          bearer_token: "",
          channel: "",
          enabled: null,
          language: "",
          kind: "slack"
        },
        general: {
          postfix_public_ip: "",
          app_language: ""
        }
      },
      languages: []
    };
  },
  computed: {
    SlackChannel: function() {
      return this.$gettext("Slack channel");
    },
    SlackChannelInputPlaceholder: function() {
      return this.$gettext("Please enter slack channel name");
    },
    SlackNotifications: function() {
      return this.$gettext("Slack notifications");
    },
    SlackNotificationsSwitchOptions: function() {
      return [this.$gettext("Yes"), this.$gettext("No")];
    },
    SlackAPItoken: function() {
      return this.$gettext("Slack API token");
    },
    SlackAPItokenPlacefolder: function() {
      return this.$gettext("Please enter api token");
    },
    PostfixPublicIP: function() {
      return this.$gettext("Postfix public IP");
    },
    EnterIpAddress: function() {
      return this.$gettext("Enter ip address");
    }
  },
  methods: {
    onGeneralSettingsSubmit(event) {
      event.preventDefault();
      let vue = this;

      const data = {
        postfixPublicIP: vue.settings.general.postfix_public_ip,
        app_language: this.$language.current
      };

      submitGeneralForm(data, true);
    },
    onNotificationSettingsSubmit(event) {
      event.preventDefault();

      const data = {
        messenger_enabled: this.MapEnabled(
          this.settings.slack_notifications.enabled
        ),
        messenger_token: this.settings.slack_notifications.bearer_token,
        messenger_kind: "slack",
        messenger_channel: this.settings.slack_notifications.channel,
        messenger_language: this.settings.slack_notifications.language
      };

      submitNotificationsSettingsForm(data);
    },
    MapEnabled(value) {
      if (false === value) {
        return "No";
      } else if (true === value) {
        return "Yes";
      } else if ("No" === value) {
        return false;
      } else if ("Yes" === value) {
        return true;
      }
      return "";
    }
  },
  mounted() {
    this.sessionInterval = this.ValidSessionCheck();

    let vue = this;
    getMetaLanguage().then(function(response) {
      vue.languages = [];
      for (let language of response.data["languages"]) {
        vue.languages.push({ text: language.key, value: language.value });
      }
    });
    getSettings().then(function(response) {
      vue.settings = response.data;
      vue.settings.slack_notifications.enabled = vue.MapEnabled(
        vue.settings.slack_notifications.enabled
      );
    });
  },
  destroyed() {
    clearInterval(this.sessionInterval);
  }
};
</script>

<style lang="less">
.settings-page .main-content {
  text-align: left;
}

.settings-page .btn-cancel {
  background: #ff5c6f33 0% 0% no-repeat padding-box;
  border: 1px solid #ff5c6f;
  border-radius: 2px;
  opacity: 0.8;
  text-align: center;
  font: normal normal bold 14px/24px Open Sans;
  letter-spacing: 0px;
  color: #820d1b;
}

.settings-page .general-save {
  background: #1d8caf33 0% 0% no-repeat padding-box;
  border: 1px solid #1d8caf;
  border-radius: 2px;
  opacity: 0.8;
  text-align: center;
  font: normal normal bold 14px/24px Open Sans;
  letter-spacing: 0px;
  color: #1d8caf;
}

.settings-page .general-save:hover,
.settings-page .general-cancel:hover {
  background: #1d8caf33 0% 0% no-repeat padding-box;
  color: #212529;
  text-decoration: none;
}

.settings-page .general-save:hover {
  background: #1d8caf33 0% 0% no-repeat padding-box;
  border: 1px solid #1d8caf;
}

.settings-page .general-cancel:hover {
  background: #ff5c6f33 0% 0% no-repeat padding-box;
  border: 1px solid #ff5c6f;
}

.settings-page [type="input"] {
  border: 1px solid #e6e7e7;
  border-radius: 5px;
  opacity: 1;
}

.settings-page #notifications-form-container {
  margin-top: 2em;
  margin-bottom: 2em;
}

.settings-page #general-form-container {
  margin-top: 2em;
  margin-bottom: 2em;
}

.settings-page .form-heading {
  margin-bottom: 0.5em;
  margin-top: 0.5em;
  font-weight: bold;
}

.settings-page .button-group {
  display: flex;
  flex-flow: row-reverse;
}

.settings-page .button-group button,
.settings-page .button-group .btn-cancel {
  width: 10%;
  margin-left: 1em;
  margin-right: 1em;
  display: flex;
  justify-content: center;
}

@media (max-width: 768px) {
  .settings-page .button-group button,
  .settings-page .button-group .btn-cancel {
    width: auto;
  }
}
</style>
