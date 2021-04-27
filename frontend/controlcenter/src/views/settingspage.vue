<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
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
        <h5 class="form-heading">
          <!-- prettier-ignore -->
          <translate>Notifications</translate>
        </h5>

        <b-form data-subsection="language" @submit="onNotificationSettingsSubmit">
          <b-form-row class="align-items-end">
            <b-col cols="6">
              <b-form-group :label="NotificationLanguage" class="notification-language">
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
                  <!-- prettier-ignore -->
                  <translate>Save</translate>
                </b-button>
              </b-form-group>
            </b-col>
          </b-form-row>
        </b-form>

        <b-form data-subsection="email" @submit="onNotificationSettingsSubmit">
          <b-form-group :label="EmailNotificationsEnabled" class="notification-disabler">
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
                <b-form-text id="mailServerPort-help-block">{{EmailNotificationDefaultPortLabel}}</b-form-text>
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
                <!-- prettier-ignore -->
                <translate>Allow insecure TLS</translate>
                &nbsp;
                <span
                  v-b-tooltip.hover
                  :title="InsecureTlsHelpText"
                >
                  <i class="fa fa-info-circle insight-help-button"></i>
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
                <!-- prettier-ignore -->
                <translate>Save</translate>
              </b-button>
              <b-button
                variant="outline-danger"
                type="button"
                @click="OnClearEmailNotificationsSettings"
                >
                <!-- prettier-ignore -->
                <translate>Reset</translate>
              </b-button>
            </div>
          </b-form-group>
        </b-form>
        
        <b-form data-subsection="slack" @submit="onNotificationSettingsSubmit">
          <b-form-group :label="SlackNotificationsEnabled" class="slack-disabler">
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
                <!-- prettier-ignore -->
                <translate>Save</translate>
              </b-button>
              <b-button
                variant="outline-danger"
                type="button"
                @click="OnClearSlackNotificationsSettings"
                >
                <!-- prettier-ignore -->
                <translate>Reset</translate>
              </b-button>
            </div>
          </b-form-group>
        </b-form>
        
        <h5 class="form-heading">
          <!-- prettier-ignore -->
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
              :placeholder="PublicURLPlaceholder"
              maxlength="255"
            ></b-form-input>
          </b-form-group>
          
          <div class="button-group">
            <b-button variant="outline-primary" type="submit">
              <!-- prettier-ignore -->
              <translate>Save</translate>
            </b-button>
            <b-button
              variant="outline-danger"
              type="button"
              @click="OnClearGeneralSettings"
              >
              <!-- prettier-ignore -->
              <translate>Reset</translate>
            </b-button>
          </div>
        </b-form>
        
        <h5 class="form-heading">
          <!-- prettier-ignore -->
          <translate>Message Detective</translate>
        </h5>

        <b-form @submit="onDetectiveSettingsSubmit">
          <b-form-group :label="DetectiveEndUsersEnabled">
            <b-form-radio-group
              class="pt-2"
              required
              v-model="settings.detective.end_users_enabled"
              :options="YesNoOptions"
            ></b-form-radio-group>
            <span class="text-warning" v-show="settings.detective.end_users_enabled">
              {{ DetectiveEndUsersHelpText }}
            </span>
          </b-form-group>
          
          <div class="button-group">
            <b-button variant="outline-primary" type="submit">
              <!-- prettier-ignore -->
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
  submitGeneralForm,
  submitNotificationsSettingsForm,
} from "@/lib/api.js";
import auth from "../mixin/auth.js";
import Vue from "vue";

export default {
  name: "settingspage",
  components: {},
  mixins: [auth],
  data() {
    return {
      settings: {
        slack_notifications: {
          bearer_token: "",
          channel: "",
          enabled: false,
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
          enabled: false,
        },
        notifications: {
          // TODO: move this to a global state
          language: "en",
        },
        general: {
          postfix_public_ip: "",
          app_language: "",
          public_url: ""
        },
        detective: {
          end_users_enabled: false
        }
      },
      languages: []
    };
  },
  computed: {
    YesNoOptions: function() {
      return [{text: this.$gettext("Yes"), value: true}, {text: this.$gettext("No"), value: false}];
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
      return this.$gettext("Username");
    },
    EmailServerPassword: function() {
      return this.$gettext("Password");
    },
    EmailServerPasswordInputPlaceholder: function() {
      return this.$gettext("Password");
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
        {text: this.$gettext("None"), value: "none"},
        {text: "STARTTLS", value: "STARTTLS"},
        {text: "TLS", value: "TLS"}
      ];
    },
    EmailNotificationDefaultPortLabel: function() {
      let options = {"STARTTLS": 587, "TLS": 465};
      let selected = options[this.settings.email_notifications.security_type]

      if (selected == undefined) {
        return ""
      }

      let translation = this.$gettext("Default: %{port}")

      return this.$gettextInterpolate(translation, {"port": selected})
    },
    EmailNotificationsAuthOptions: function() {
      return [{text: this.$gettext("No Authentication"), value: "none"}, {text: this.$gettext("Password"), value: "password"}];
    },
    EmailFieldRequired: function() {
      return this.settings.email_notifications.enabled
        || this.settings.email_notifications.auth_method != "none"
        || this.settings.email_notifications.server_port != "0";
    },
    EmailPortFieldRequired: function() {
      return this.settings.email_notifications.enabled || this.settings.email_notifications.auth_method != "none";
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
      return this.$gettext("Please enter API token");
    },
    SlackFieldRequired: function() {
      return this.settings.slack_notifications.enabled;
    },
    PostfixPublicIP: function() {
      return this.$gettext("Postfix public IP");
    },
    PublicURL: function() {
      return this.$gettext("Public URL");
    },
    EnterIpAddress: function() {
      return this.$gettext("Enter IP address");
    },
    PublicURLPlaceholder: function() {
      return this.$gettext("Enter Public URL");
    },
    InsecureTlsHelpText() {
      return this.$gettext("Certificates will be used but not validated, allowing insecure connections");
    },
    DetectiveEndUsersEnabled() {
      return this.$gettext("Allow unauthenticated end users to use Message Detective");
    },
    DetectiveEndUsersHelpText() {
      return this.$gettext("Enabling this feature may leak data");  // TODO: better explanation
    },
  },
  methods: {
    RefreshSettings() {
      let vue = this;
      
      getSettings().then(function(response) {
        vue.settings = response.data;
        if (vue.settings.notifications.language === "") {
          vue.settings.notifications.language = "en";
        }
      });
    },
    OnClearEmailNotificationsSettings(event) {
      event.preventDefault();
      let vue = this;
      
      if (!confirm(Vue.prototype.$gettext("Reset email notification settings?")))
        return;
      
      clearSettings('notification', 'email').then(function() {
        vue.RefreshSettings();
      });
    },
    OnClearSlackNotificationsSettings(event) {
      event.preventDefault();
      let vue = this;
      
      if (!confirm(Vue.prototype.$gettext("Reset slack notification settings?")))
        return;
      
      clearSettings('notification', 'slack').then(function() {
        vue.RefreshSettings();
      });
    },
    OnClearGeneralSettings(event) {
      event.preventDefault();
      let vue = this;
      
      if (!confirm(Vue.prototype.$gettext("Reset general settings?")))
        return;
      
      clearSettings("general").then(function() {
        vue.RefreshSettings();
      });
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
      
      let subsection = event.target.getAttribute('data-subsection');
      
      const data = {
        slack: {
          messenger_enabled: this.settings.slack_notifications.enabled,
          messenger_token: this.settings.slack_notifications.bearer_token,
          messenger_channel: this.settings.slack_notifications.channel,
        },
        language: {
          notification_language: this.settings.notifications.language,
        },
        email: {
          email_notification_server_name: this.settings.email_notifications.server_name,
          email_notification_skip_cert_check: this.settings.email_notifications.skip_cert_check,
          email_notification_port: this.settings.email_notifications.server_port,
          email_notification_username: this.settings.email_notifications.username,
          email_notification_password: this.settings.email_notifications.password,
          email_notification_sender: this.settings.email_notifications.sender,
          email_notification_recipients: this.settings.email_notifications.recipients,
          email_notification_security_type: this.settings.email_notifications.security_type,
          email_notification_auth_method: this.settings.email_notifications.auth_method,
          email_notification_enabled: this.settings.email_notifications.enabled,
        }
      }[subsection];

      let trackingInfo = {"SlackEnabled": this.settings.slack_notifications.enabled, "EmailEnabled": this.settings.email_notifications.enabled};

      submitNotificationsSettingsForm(data, trackingInfo);
    },
    onDetectiveSettingsSubmit(event) {
      event.preventDefault();
      
      const data = {
        detective_end_users_enabled: this.settings.detective.end_users_enabled,
      };

      let trackingInfo = {"EndUsersEnabled": this.settings.detective.end_users_enabled};

      submitDetectiveSettingsForm(data, trackingInfo);
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
    vue.RefreshSettings();
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

.form-row
, .form-container form label {
  font-size: 16px;
}

.form-group input
, .form-group select {
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
  border-color: #1D8CAF;
  background-color: #1D8CAF;
}

form .form-control:focus
, form .custom-select:focus {
  border-color: #32ABE4;
  box-shadow: 0 0 0 0.2rem #DCF1FB;
}

@media (max-width: 768px) {
  .settings-page .button-group button {
    width: auto;
  }
}
</style>
