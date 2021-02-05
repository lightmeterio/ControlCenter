<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <panel-page>
    <div id="login-page">
      <h2>
        <!-- prettier-ignore -->
        <translate>Login</translate>
      </h2>
      <div class="field-group">
        <b-form @submit="onSubmit">
          <b-form-group>
            <b-form-input
              name="email"
              id="email"
              v-model="form.email"
              type="email"
              required
              aria-describedby="emailHelp"
              :placeholder="EmailInputPlaceholder"
              maxlength="255"
            ></b-form-input>
            <b-input-group id="show_hide_password">
              <b-form-input
                name="password"
                id="password"
                v-model="form.password"
                required
                aria-describedby="passwordHelp"
                :placeholder="PasswordInputPlaceholder"
                type="password"
                maxlength="255"
              ></b-form-input>
              <div class="input-group-addon" v-on:click="onTogglePasswordShow">
                <a href=""><i class="fa fa-eye" aria-hidden="true"></i></a>
              </div>
            </b-input-group>
            <p class="align-right">
              <small
                ><a
                  target="_blank"
                  href="https://gitlab.com/lightmeter/controlcenter#password-reset"
                >
                  <!-- prettier-ignore -->
                  <translate>Forgot password?</translate>
                </a></small
              >
            </p>
          </b-form-group>
          <b-button variant="primary" class="w-100" type="submit">
            <!-- prettier-ignore -->
            <translate>Login</translate>
          </b-button>
        </b-form>
      </div>
    </div>
  </panel-page>
</template>

<script>
import { submitLoginForm, submitGeneralForm } from "../lib/api.js";
import { togglePasswordShow } from "../lib/util.js";
import { mapState } from "vuex";

export default {
  name: "login",
  components: {},
  data() {
    return {
      form: {
        email: "",
        password: ""
      }
    };
  },
  computed: {
    PasswordInputPlaceholder: function() {
      return this.$gettext("Password");
    },
    EmailInputPlaceholder: function() {
      return this.$gettext("Email");
    },
    ...mapState(["language"])
  },
  methods: {
    onSubmit(event) {
      event.preventDefault();
      let vue = this;
      const callback = () => {
        const data = {
          app_language: vue.language
        };
        submitGeneralForm(data, false);

        vue.$router.push({ name: "index" });
      };
      submitLoginForm(this.form, callback);
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
#login-page .btn-primary:hover {
  color: #fff;
  background-color: #0069d9;
  border-color: #0062cc;
}
</style>
