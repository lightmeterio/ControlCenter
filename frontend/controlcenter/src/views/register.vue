<template>
  <panel-page>
    <div id="registration-page">
      <h2>
        Welcome
        <!--{{translate "Welcome"}}-->
      </h2>
      <p class="align-left">
        Please create a new administrator account - this is necessary to login
        <!--{{translate "Please create a new administrator account - this is necessary to login."}}-->
        <a href="https://gitlab.com/lightmeter/controlcenter#upgrade"
          >Get help<!--{{translate "Get help"}}--></a
        >
        to avoid repeating this step if you've done it before
        <!--{{translate "to avoid repeating this step if you've done it before"}}-->
      </p>

      <div class="field-group">
        <h4>
          User details
          <!--{{translate "User details"}}-->
        </h4>
        <b-form @submit="onSubmit">
          <b-form-group>
            <b-form-input
              name="name"
              id="name"
              v-model="form.name"
              type="text"
              required
              aria-describedby="nameHelp"
              placeholder="Name"
              maxlength="255"
            ></b-form-input>
            <b-form-input
              name="email"
              id="email"
              v-model="form.email"
              type="email"
              required
              aria-describedby="emailHelp"
              placeholder="Email"
              maxlength="255"
            ></b-form-input>
            <b-input-group id="show_hide_password">
              <b-form-input
                name="password"
                id="password"
                v-model="form.password"
                type="password"
                required
                aria-describedby="passwordHelp"
                placeholder="Password"
                maxlength="255"
              ></b-form-input>
              <div class="input-group-addon" v-on:click="onTogglePasswordShow">
                <a href=""><i class="fa fa-eye" aria-hidden="true"></i></a>
              </div>

              <div class="input-group">
                <select
                  required
                  v-model="form.email_kind"
                  class="form-control custom-select"
                  name="email_kind"
                  id="email_kind"
                >
                  <option value="" selected disabled
                    >Most of my mail is…<!--{{translate "Most of my mail is…"}}--></option
                  >
                  <option value="direct"
                    >Direct (personal, office, one-to-one)<!--{{translate "Direct (personal, office, one-to-one)"}}--></option
                  >
                  <option value="transactional"
                    >Transactional (notifications, apps)<!--{{translate "Transactional (notifications, apps)"}}--></option
                  >
                  <option value="marketing"
                    >Marketing (newsletters, adverts)<!--{{translate "Marketing (newsletters, adverts)"}}--></option
                  >
                </select>
                <div class="input-group-append">
                  <button
                    class="btn btn-outline-secondary"
                    type="button"
                    data-toggle="tooltip"
                    data-placement="top"
                    v-b-tooltip.hover
                    title="Different types of mail perform differently. This helps show the most relevant information."
                  >
                    <!--{{translate `Different types of mail perform differently. This helps show the most relevant information.`}}-->
                    <i class="far fa-question-circle"></i>
                  </button>
                </div>
              </div>

              <b-form-checkbox
                id="subscribe_newsletter"
                v-model="form.subscribe_newsletter"
                name="subscribe_newsletter"
                value="on"
                unchecked-value="off"
                class="custom-form-check-label"
              >
                Monthly newsletter
              </b-form-checkbox>
            </b-input-group>
          </b-form-group>
          <b-button variant="primary" class="w-100" type="submit"
            >Register</b-button
          ><!--{{translate `Register`}}-->
        </b-form>
        <div class="card info" v-if="tracking()">
          <div class="card-body">
            <h5 class="card-title">
              <i class="fa fa-info-circle"></i>
              Telemetry enabled
              <!--{{ translate`Telemetry enabled` }}-->
            </h5>
            <p class="card-text">
              Feature usage data is shared with a private Open Source analytics
              system to improve your experience and may be
              <a href="https://lightmeter.io/privacy-policy/">disabled</a> at
              any time
              <!-- {{translate `Feature usage data is shared with a private Open Source
              analytics system to improve your experience and may be
              <a href="https://lightmeter.io/privacy-policy/">disabled</a> at any
              time`}} -->
            </p>
          </div>
        </div>
      </div>
    </div>
  </panel-page>
</template>

<script>
import { submitRegisterForm } from "../lib/api.js";
import { togglePasswordShow } from "../lib/util.js";

export default {
  name: "register",
  components: {},
  data() {
    return {
      form: {
        email: "",
        password: "",
        name: ``,
        subscribe_newsletter: null,
        email_kind: ""
      }
    };
  },
  methods: {
    onSubmit(event) {
      event.preventDefault();
      let vue = this;

      let settingsData = {
        email: this.form.email,
        email_kind: this.form.email_kind
      };

      if (this.form.subscribe_newsletter !== null) {
        settingsData.subscribe_newsletter = this.form.subscribe_newsletter;
      }

      const registrationData = {
        email: this.form.email,
        name: this.form.name,
        password: this.form.password
      };

      const redirect = () => {
        vue.$router.push({ name: "index" });
      };

      submitRegisterForm(registrationData, settingsData, redirect);
    },
    tracking() {
      if (window.doNotTrack || navigator.doNotTrack || navigator.msDoNotTrack) {
        if (
            (window.doNotTrack == 1 ||
          navigator.doNotTrack == "yes" ||
          navigator.doNotTrack == 1 ||
          navigator.msDoNotTrack == 1)
        ) {
          return false;
        }
      }
      return true;
    },
    onTogglePasswordShow(event) {
      event.preventDefault();
      togglePasswordShow(event);
    }
  },
  mounted() {
    const el = document.body;
    el.classList.add("login-gradient");
  },
  destroyed() {
    const el = document.body;
    el.classList.remove("login-gradient");
  }
};
</script>

<style lang="less">
#registration-page .card {
  margin-top: 1em;
  text-align: left;
}

#registration-page .card .card-text {
  color: #00689d;
  font-size: 12px;
}

#registration-page .card.info {
  background: #daebf4;
  border: none;
}

#registration-page .card.info .card-title {
  font-size: 12px;
  letter-spacing: 0px;
  font-weight: bold;
  margin-bottom: 0.8em;
  color: #00689d;
}

#registration-page .card.info .card-body {
  padding: 0.8em;
}

#registration-page .card .fa {
  padding-right: 0.8em;
}
</style>
