<template>
  <panel-page>
    <h2>
      Login
      <!--{{translate "Login"}}-->
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
            placeholder="Email"
            maxlength="255"
          ></b-form-input>
          <b-input-group id="show_hide_password">
            <b-form-input
              name="password"
              id="password"
              v-model="form.password"
              required
              aria-describedby="passwordHelp"
              placeholder="Password"
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
                >Forgot password?</a
              ></small
            >
          </p>
          <!--{{translate "Forgot password?"}}-->
        </b-form-group>
        <b-button variant="primary" class="w-100" type="submit">Login</b-button
        ><!--{{translate `Login`}}-->
      </b-form>
    </div>
  </panel-page>
</template>

<script>
import { submitLoginForm } from "../lib/api.js";
import { togglePasswordShow } from "../lib/util.js";

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

  methods: {
    onSubmit(event) {
      event.preventDefault();
      let vue = this;
      const redirect = () => {
        vue.$router.push({ name: "index" });
      };
      submitLoginForm(this.form, redirect);
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

<style lang="less"></style>
