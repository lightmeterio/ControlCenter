<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="settings-page d-flex flex-column min-vh-100">
    <mainheader></mainheader>

    <b-container id="settings" class="main-content">
      <h2 class="form-heading">
        <translate>Settings</translate>
      </h2>
      <div class="form-container">
        <h5 class="form-heading">
          <translate>Notifications</translate>
        </h5>

        <b-form
          data-subsection="language"
          @submit="onNotificationSettingsSubmit"
        >
          <b-form-row class="align-items-end">
            <b-col cols="6">
              <b-form-group
                :label="NotificationLanguage"
                class="notification-language"
              >
                <b-form-select
                  class="pt-2"
                  required
                  v-model="settings.notifications.language"
                  :options="languages"
                  stacked
                ></b-form-select>
              </b-form-group>
            </b-col>
            <b-col cols="6">
              <b-form-group>
                <b-button variant="outline-primary" type="submit">
                  <translate>Save</translate>
                </b-button>
              </b-form-group>
            </b-col>
          </b-form-row>
        </b-form>

        <b-form data-subsection="email" @submit="onNotificationSettingsSubmit">
          <b-form-group
            :label="EmailNotificationsEnabled"
            class="notification-disabler"
          >
            <b-form-radio-group
              class="pt-2"
              required
              v-model="settings.email_notifications.enabled"
              :options="YesNoOptions"
            ></b-form-radio-group>
            <b-form-row>
              <b-col cols="6">
                <b-form-group
                  class="mail-server-name"
                  :label="EmailServerName"
                  label-for="mailServerName"
                >
                  <b-form-input
                    name="mail_server_name"
                    id="mailServerName"
                    v-model="settings.email_notifications.server_name"
                    :placeholder="EmailServerNameInputPlaceholder"
                    maxlength="255"
                    :required="EmailFieldRequired"
                  ></b-form-input>
                </b-form-group>
              </b-col>
              <b-col cols="6">
                <b-form-row>
                  <b-col cols="6">
                    <b-form-group
                      class="mail-server-port"
                      :label="EmailServerPort"
                      label-for="mailServerPort"
                    >
                      <b-form-input
                        type="number"
                        name="mail_server_port"
                        id="mailServerPort"
                        v-model="settings.email_notifications.server_port"
                        maxlength="255"
                        :required="EmailPortFieldRequired"
                        min="0"
                        max="65536"
                      ></b-form-input>
                    </b-form-group>
                  </b-col>
                  <b-col class="align-self-center" cols="6">
                    <b-form-text id="mailServerPort-help-block">{{
                      EmailNotificationDefaultPortLabel
                    }}</b-form-text>
                  </b-col>
                </b-form-row>
              </b-col>
            </b-form-row>
            <b-form-row>
              <b-col cols="6">
                <b-form-group
                  class="mail-server-auth-method"
                  :label="EmailServerSecurityType"
                  label-for="mailServerSecurityType"
                >
                  <b-form-select
                    name="mail_server_security_type"
                    id="mailServerSecurityType"
                    v-model="settings.email_notifications.security_type"
                    :options="EmailNotificationsSecurityTypeOptions"
                  ></b-form-select>
                </b-form-group>
              </b-col>
              <b-col cols="6">
                <b-form-group
                  class="mail-server-auth-method"
                  :label="EmailServerAuthMethod"
                  label-for="mailServerAuthMethod"
                >
                  <b-form-select
                    name="mail_server_auth_method"
                    id="mailServerAuthMethod"
                    v-model="settings.email_notifications.auth_method"
                    :options="EmailNotificationsAuthOptions"
                  ></b-form-select>
                </b-form-group>
              </b-col>
            </b-form-row>

            <b-form-group
              class="mail-server-auth-skip-cert-check"
              label-for="mailServerSkipCertCheck"
            >
              <b-form-checkbox
                name="mail_server_skip_cert_check"
                id="mailServerSkipCertCheck"
                v-model="settings.email_notifications.skip_cert_check"
              >
                <translate>Allow insecure TLS</translate>
                &nbsp;
                <span v-b-tooltip.hover :title="InsecureTlsHelpText">
                  <i class="fa fa-info-circle lm-info-circle-grayblue"></i>
                </span>
              </b-form-checkbox>
            </b-form-group>

            <b-form-group
              class="mail-server-auth-username"
              :label="EmailServerUsername"
              label-for="mailServerUsername"
            >
              <b-form-input
                name="mail_server_username"
                id="mailServerUsername"
                v-model="settings.email_notifications.username"
                :placeholder="EmailServerUsernameInputPlaceholder"
                maxlength="255"
                :required="EmailAuthenticationIsRequired"
                :disabled="!EmailAuthenticationIsRequired"
              ></b-form-input>
            </b-form-group>

            <b-form-group
              class="mail-server-auth-password"
              :label="EmailServerPassword"
              label-for="mailServerPassword"
            >
              <b-form-input
                name="mail_server_password"
                id="mailServerPassword"
                v-model="settings.email_notifications.password"
                :placeholder="EmailServerPasswordInputPlaceholder"
                maxlength="255"
                :required="EmailAuthenticationIsRequired"
                type="password"
                :disabled="!EmailAuthenticationIsRequired"
              ></b-form-input>
            </b-form-group>

            <b-form-group
              class="mail-server-auth-sender"
              :label="EmailServerSender"
              label-for="mailServerSender"
            >
              <b-form-input
                name="mail_server_sender"
                id="mailServerSender"
                v-model="settings.email_notifications.sender"
                :placeholder="EmailServerSenderInputPlaceholder"
                maxlength="255"
                :required="EmailFieldRequired"
              ></b-form-input>
            </b-form-group>

            <b-form-group
              class="mail-server-auth-recipients"
              :label="EmailServerRecipients"
              label-for="mailServerRecipients"
            >
              <b-form-input
                name="mail_server_recipients"
                id="mailServerRecipients"
                v-model="settings.email_notifications.recipients"
                :placeholder="EmailServerRecipientsInputPlaceholder"
                maxlength="255"
                :required="EmailFieldRequired"
              ></b-form-input>
            </b-form-group>
            <div class="button-group">
              <b-button variant="outline-primary" type="submit">
                <translate>Save</translate>
              </b-button>
              <b-button
                variant="outline-danger"
                type="button"
                @click="OnClearEmailNotificationsSettings"
              >
                <translate>Reset</translate>
              </b-button>
            </div>
          </b-form-group>
        </b-form>

        <b-form data-subsection="slack" @submit="onNotificationSettingsSubmit">
          <b-form-group
            :label="SlackNotificationsEnabled"
            class="slack-disabler"
          >
            <b-form-radio-group
              class="pt-2"
              required
              v-model="settings.slack_notifications.enabled"
              :options="YesNoOptions"
            ></b-form-radio-group>

            <b-form-group
              class="slack-channel"
              :label="SlackChannel"
              label-for="slackChannel"
            >
              <b-form-input
                name="messenger_channel"
                id="slackChannel"
                v-model="settings.slack_notifications.channel"
                :placeholder="SlackChannelInputPlaceholder"
                maxlength="255"
                :required="SlackFieldRequired"
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
                :placeholder="SlackAPItokenPlacefolder"
                maxlength="255"
                :required="SlackFieldRequired"
              ></b-form-input>
            </b-form-group>
            <div class="button-group">
              <b-button variant="outline-primary" type="submit">
                <translate>Save</translate>
              </b-button>
              <b-button
                variant="outline-danger"
                type="button"
                @click="OnClearSlackNotificationsSettings"
              >
                <translate>Reset</translate>
              </b-button>
            </div>
          </b-form-group>
        </b-form>

        <h5 class="form-heading">
          <translate>General</translate>
        </h5>

        <b-form @submit="onGeneralSettingsSubmit">
          <b-form-group
            class="postfixPublicIP"
            :label="PostfixPublicIP"
            label-for="postfixPublicIP"
          >
            <b-form-input
              name="postfix_public_ip"
              id="postfixPublicIP"
              v-model="settings.general.postfix_public_ip"
              required
              :placeholder="EnterIpAddress"
              maxlength="255"
            ></b-form-input>
            <b-form-text>{{ PublicIPHelpText }}</b-form-text>
          </b-form-group>

          <b-form-group
            class="publicURL"
            :label="PublicURL"
            label-for="publicURL"
          >
            <b-form-input
              name="publicURL"
              id="publicURL"
              v-model="settings.general.public_url"
              required
              placeholder="http://lightmeter.mywebsite.com"
              maxlength="255"
            ></b-form-input>
          </b-form-group>

          <div class="button-group">
            <b-button variant="outline-primary" type="submit">
              <translate>Save</translate>
            </b-button>
            <b-button
              variant="outline-danger"
              type="button"
              @click="OnClearGeneralSettings"
            >
              <translate>Reset</translate>
            </b-button>
          </div>
        </b-form>

        <h5 class="form-heading">
          <translate>Message Detective</translate>
        </h5>

        <b-form @submit="onDetectiveSettingsSubmit">
          <p
            class="pt-2"
            render-html="true"
            v-translate="{
              openLink: openDetectiveLink,
              closeLink: closeDetectiveLink
            }"
          >
            See %{openLink}documentation%{closeLink} for details
          </p>
          <b-form-group :label="DetectiveEndUsersEnabled">
            <b-form-radio-group
              class="pt-2"
              required
              v-model="settings.detective.end_users_enabled"
              :options="YesNoOptions"
            ></b-form-radio-group>
            <div v-show="settings.detective.end_users_enabled">
              <p class="message-detective-settings-info">
                {{ DetectiveEndUsersHelpText }}
              </p>
              <div class="message-detective-url-area">
                <a :href="endUsersURL">{{ endUsersURL }}</a>
              </div>
            </div>
          </b-form-group>

          <div class="button-group">
            <b-button variant="outline-primary" type="submit">
              <translate>Save</translate>
            </b-button>
          </div>
        </b-form>

        <h5 class="form-heading">
          <translate>Insights</translate>
        </h5>

        <b-form @submit="onInsightsSettingsSubmit">
          <b-form-group
            :label="LabelBounceRateThreshold"
            :description="DescriptionBounceRateThreshold"
            label-cols-sm="4"
            label-cols-lg="5"
            content-cols-sm
            content-cols-lg
          >
            <b-input-group append="%">
              <b-form-input
                required
                v-model="settings.insights.bounce_rate_threshold"
                type="number"
                name="bounce_rate_threshold"
                min="0"
                max="100"
              ></b-form-input>
            </b-input-group>
          </b-form-group>

          <div class="button-group">
            <b-button variant="outline-primary" type="submit">
              <translate>Save</translate>
            </b-button>
          </div>
        </b-form>
      </div>
    </b-container>
    <mainfooter></mainfooter>
  </div>
</template>

<script>
import {
  clearSettings,
  getMetaLanguage,
  getSettings,
  submitDetectiveSettingsForm,
  submitInsightsSettingsForm,
  submitGeneralForm,
  submitNotificationsSettingsForm
} from "@/lib/api.js";
import { trackEvent } from "@/lib/util";
import auth from "../mixin/auth.js";
import shared_texts from "../mixin/shared_texts.js";
import Vue from "vue";

export default {
  name: "settingspage",
  components: {},
  mixins: [auth, shared_texts],
  data() {
    return {
      settings: {
        slack_notifications: {
          bearer_token: "",
          channel: "",
          enabled: false
        },
        email_notifications: {
          server_name: "",
          skip_cert_check: false,
          server_port: 0,
          sender: "",
          recipients: "",
          security_type: "none",
          auth_method: "none",
          username: "",
          password: "",
          enabled: false
        },
        notifications: {
          // TODO: move this to a global state
          language: "en"
        },
        general: {
          postfix_public_ip: "",
          app_language: "",
          public_url: ""
        },
        detective: {
          end_users_enabled: false
        },
        insights: {
          bounce_rate_threshold: 30
        }
      },
      prev_settings: {},
      languages: [],
      endUsersURL:
        window.location.origin + window.location.pathname + "#/searchmessage"
    };
  },
  computed: {
    YesNoOptions: function() {
      return [
        { text: this.$gettext("Yes"), value: true },
        { text: this.$gettext("No"), value: false }
      ];
    },
    NotificationLanguage: function() {
      return this.$gettext("Language");
    },
    EmailNotificationsEnabled: function() {
      return this.$gettext("Email Notifications");
    },
    EmailServerName: function() {
      return this.$gettext("Server Name");
    },
    EmailServerNameInputPlaceholder: function() {
      return this.$gettext("Name or IP address");
    },
    EmailServerPort: function() {
      return this.$gettext("Port");
    },
    EmailServerSecurityType: function() {
      return this.$gettext("Connection Security Type");
    },
    EmailServerAuthMethod: function() {
      return this.$gettext("Authentication Method");
    },
    EmailServerUsername: function() {
      return this.$gettext("Username");
    },
    EmailServerUsernameInputPlaceholder: function() {
      return "⬤⬤⬤⬤⬤⬤";
    },
    EmailServerPassword: function() {
      return this.$gettext("Password");
    },
    EmailServerPasswordInputPlaceholder: function() {
      return "⬤⬤⬤⬤⬤⬤";
    },
    EmailServerSender: function() {
      return this.$gettext("Sender");
    },
    EmailServerSenderInputPlaceholder: function() {
      return this.$gettext("Used in the From: header");
    },
    EmailServerRecipients: function() {
      return this.$gettext("Recipients");
    },
    EmailServerRecipientsInputPlaceholder: function() {
      return this.$gettext("Used in the To: header");
    },
    EmailNotificationsSecurityTypeOptions: function() {
      return [
        { text: this.$gettext("None"), value: "none" },
        { text: "STARTTLS", value: "STARTTLS" },
        { text: "TLS", value: "TLS" }
      ];
    },
    EmailNotificationDefaultPortLabel: function() {
      let options = { STARTTLS: 587, TLS: 465 };
      let selected = options[this.settings.email_notifications.security_type];

      if (selected == undefined) {
        return "";
      }

      let translation = this.$gettext("Default: %{port}");

      return this.$gettextInterpolate(translation, { port: selected });
    },
    EmailNotificationsAuthOptions: function() {
      return [
        { text: this.$gettext("No Authentication"), value: "none" },
        { text: this.$gettext("Password"), value: "password" }
      ];
    },
    EmailFieldRequired: function() {
      return (
        this.settings.email_notifications.enabled ||
        this.settings.email_notifications.auth_method != "none" ||
        this.settings.email_notifications.server_port != "0"
      );
    },
    EmailPortFieldRequired: function() {
      return (
        this.settings.email_notifications.enabled ||
        this.settings.email_notifications.auth_method != "none"
      );
    },
    EmailAuthenticationIsRequired: function() {
      return this.settings.email_notifications.auth_method != "none";
    },
    SlackChannel: function() {
      return this.$gettext("Slack channel");
    },
    SlackChannelInputPlaceholder: function() {
      return this.$gettext("Please enter Slack channel name");
    },
    SlackNotificationsEnabled: function() {
      return this.$gettext("Slack Notifications");
    },
    SlackAPItoken: function() {
      return this.$gettext("Slack API token");
    },
    SlackAPItokenPlacefolder: function() {
      return "⬤⬤⬤⬤⬤⬤";
    },
    SlackFieldRequired: function() {
      return this.settings.slack_notifications.enabled;
    },
    PostfixPublicIP: function() {
      return this.$gettext("Mail server public IP");
    },
    PublicURL: function() {
      return this.$gettext("Lightmeter public URL");
    },
    EnterIpAddress: function() {
      return this.$gettext("Enter IP address");
    },
    InsecureTlsHelpText() {
      return this.$gettext(
        "Certificates will be used but not validated, allowing insecure connections"
      );
    },
    DetectiveEndUsersEnabled() {
      return this.$gettext(
        "Enable public access to the Message Detective search page"
      );
    },
    LabelBounceRateThreshold() {
      return this.$gettext("Bounce Rate Threshold");
    },
    DescriptionBounceRateThreshold() {
      return this.$gettext(
        "If the ratio of bounced emails to sent emails gets above this percentage, an insight is generated."
      );
    },
    DetectiveEndUsersHelpText() {
      return this.$gettext(
        "Anyone with the link below can check email delivery outcomes (requires both 'from' and 'to' email addresses, searches are rate-limited)"
      );
    },
    openDetectiveLink() {
      return `<a href="https://gitlab.com/lightmeter/controlcenter/-/tree/master#message-detective">`;
    },
    closeDetectiveLink() {
      return `</a>`;
    }
  },
  methods: {
    RefreshSettings(fillVueSettings, checkSettingsChanges) {
      let vue = this;

      getSettings().then(function(response) {
        let new_settings = response.data;

        if (checkSettingsChanges) {
          if (
            new_settings.email_notifications.enabled !=
            vue.prev_settings.email_notifications.enabled
          ) {
            trackEvent(
              "SaveNotificationSettingsEmail",
              new_settings.email_notifications.enabled ? "enabled" : "disabled"
            );
          }
          if (
            new_settings.slack_notifications.enabled !=
            vue.prev_settings.slack_notifications.enabled
          ) {
            trackEvent(
              "SaveNotificationSettingsSlack",
              new_settings.slack_notifications.enabled ? "enabled" : "disabled"
            );
          }
        }

        if (new_settings.notifications.language === "") {
          new_settings.notifications.language = "en";
        }

        vue.prev_settings = JSON.parse(JSON.stringify(new_settings));

        if (fillVueSettings) {
          vue.settings = new_settings;
          if (!vue.settings.general.public_url)
            vue.settings.general.public_url =
              window.location.origin + window.location.pathname;
        }
      });
    },
    OnClearEmailNotificationsSettings(event) {
      event.preventDefault();

      if (
        confirm(Vue.prototype.$gettext("Reset email notification settings?"))
      ) {
        clearSettings("notification", "email").then(
          this.UpdateAndNotifySettings
        );
      }
    },
    OnClearSlackNotificationsSettings(event) {
      event.preventDefault();

      if (
        confirm(Vue.prototype.$gettext("Reset slack notification settings?"))
      ) {
        clearSettings("notification", "slack").then(
          this.UpdateAndNotifySettings
        );
      }
    },
    OnClearGeneralSettings(event) {
      event.preventDefault();

      if (confirm(Vue.prototype.$gettext("Reset general settings?"))) {
        clearSettings("general").then(this.UpdateAndNotifySettings);
      }
    },
    onGeneralSettingsSubmit(event) {
      event.preventDefault();
      let vue = this;

      const data = {
        postfix_public_ip: vue.settings.general.postfix_public_ip,
        app_language: this.$language.current,
        public_url: vue.settings.general.public_url
      };

      submitGeneralForm(data, true);
    },
    onNotificationSettingsSubmit(event) {
      event.preventDefault();

      let subsection = event.target.getAttribute("data-subsection");

      const data = {
        slack: {
          messenger_enabled: this.settings.slack_notifications.enabled,
          messenger_token: this.settings.slack_notifications.bearer_token,
          messenger_channel: this.settings.slack_notifications.channel
        },
        language: {
          notification_language: this.settings.notifications.language
        },
        email: {
          email_notification_server_name: this.settings.email_notifications
            .server_name,
          email_notification_skip_cert_check: this.settings.email_notifications
            .skip_cert_check,
          email_notification_port: this.settings.email_notifications
            .server_port,
          email_notification_username: this.settings.email_notifications
            .username,
          email_notification_password: this.settings.email_notifications
            .password,
          email_notification_sender: this.settings.email_notifications.sender,
          email_notification_recipients: this.settings.email_notifications
            .recipients,
          email_notification_security_type: this.settings.email_notifications
            .security_type,
          email_notification_auth_method: this.settings.email_notifications
            .auth_method,
          email_notification_enabled: this.settings.email_notifications.enabled
        }
      }[subsection];

      submitNotificationsSettingsForm(data).then(
        this.NotifySettingsWithoutUpdating
      );
    },
    InitializeSettings() {
      this.RefreshSettings(true, false);
    },
    UpdateAndNotifySettings() {
      this.RefreshSettings(true, true);
    },
    NotifySettingsWithoutUpdating() {
      this.RefreshSettings(false, true);
    },
    onDetectiveSettingsSubmit(event) {
      event.preventDefault();

      const data = {
        detective_end_users_enabled: this.settings.detective.end_users_enabled
      };

      submitDetectiveSettingsForm(
        data,
        this.settings.detective.end_users_enabled
      );
    },
    onInsightsSettingsSubmit(event) {
      event.preventDefault();

      submitInsightsSettingsForm({
        bounce_rate_threshold: this.settings.insights.bounce_rate_threshold
      });
    }
  },
  mounted() {
    let vue = this;

    getMetaLanguage().then(function(response) {
      vue.languages = [];
      for (let language of response.data["languages"]) {
        vue.languages.push({ text: language.key, value: language.value });
      }
    });

    this.InitializeSettings();
  }
};
</script>

<style lang="less">
.settings-page .main-content {
  text-align: left;
  max-width: 568px;
  margin-bottom: 1rem; /* FIXME: this will be redundant when bootstrap rows are used more extensively */
}

h2.form-heading {
  font-size: 32px;
}

h5.form-heading {
  font-size: 18px;
}

.form-row,
.form-container form label {
  font-size: 16px;
}

.form-group input,
.form-group select {
  font-size: 16px;
}

.form-container form legend {
  font-size: 15px;
  font-weight: bold;
}

form fieldset.form-group {
  margin: 1rem 0;
}

form .form-group {
  margin: 0.5rem 0;
}

.settings-page [type="input"] {
  border: 1px solid #e6e7e7;
  border-radius: 5px;
  opacity: 1;
}

.settings-page .form-heading {
  margin-bottom: 0.5em;
  margin-top: 0.5em;
  font-weight: bold;
}

.settings-page .button-group {
  display: flex;
  flex-flow: row-reverse;

  button {
    width: 20%;
    margin-left: 1em;
    margin-right: 1em;
  }
}

.custom-control .custom-control-input:checked ~ .custom-control-label::before {
  border-color: #1d8caf;
  background-color: #1d8caf;
}

form .form-control:focus,
form .custom-select:focus {
  border-color: #32abe4;
  box-shadow: 0 0 0 0.2rem #dcf1fb;
}

@media (max-width: 768px) {
  .settings-page .button-group button {
    width: auto;
  }
}

.message-detective-settings-info {
  margin-top: 20px;
}

.message-detective-url-area {
  padding: 20px;
  background-color: #f8f8f8;
}
</style>
