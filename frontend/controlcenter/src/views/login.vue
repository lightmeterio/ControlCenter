<template>
  <div class="login-page">
    <b-container>
      <b-row class="justify-content-center vh-93">
        <b-col sm="5" class="align-self-center">
          <div class="panel panel-default mx-auto">
            <div class="panel-body">
              <b-row class="justify-content-center">
                <b-col lg="10">
                  <img
                    class="logo w-100"
                    src="@/assets/logo-color-120.png"
                    alt="Lightmeter logo"
                  />
                </b-col>
              </b-row>

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
                        maxlength="255"
                        type="password"
                      ></b-form-input>
                      <div class="input-group-addon" v-on:click="Password">
                        <a href=""
                          ><i class="fa fa-eye" aria-hidden="true"></i
                        ></a>
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
                  <b-button variant="primary" class="w-100" type="submit"
                    >Login</b-button
                  ><!--{{translate `Login`}}-->
                </b-form>
              </div>
            </div>
          </div>
        </b-col>
      </b-row>
    </b-container>

    <footer class="mt-auto">
      <b-container class="container">
        Made with
        <svg
          class="bi bi-heart-fill"
          width="1em"
          height="1em"
          viewBox="0 0 16 16"
          fill="currentColor"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            fill-rule="evenodd"
            d="M8 1.314C12.438-3.248 23.534 4.735 8 15-7.534 4.736 3.562-3.248 8 1.314z"
          />
        </svg>
        by
        <a
          href="https://lightmeter.io"
          target="_blank"
          data-toggle="tooltip"
          data-placement="top"
          title="Lightmeter website"
          >Open Source professionals</a
        >
      </b-container>
    </footer>
  </div>
</template>

<script>
import { submitLoginForm } from "../lib/api.js";

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
    onSubmit(evt) {
      evt.preventDefault();
      submitLoginForm(this.form);
    },
    Password(event) {
      event.preventDefault();
      let attrValue = document
        .querySelector("#show_hide_password input")
        .getAttribute("type");
      if (attrValue === "text") {
        document
          .querySelector("#show_hide_password input")
          .setAttribute("type", "password");
        let i = document.querySelector("#show_hide_password svg");
        i.classList.add("fa-eye");
        i.classList.remove("fa-eye-slash");
      } else if (attrValue === "password") {
        document
          .querySelector("#show_hide_password input")
          .setAttribute("type", "text");
        let i = document.querySelector("#show_hide_password svg");
        i.classList.remove("fa-eye");
        i.classList.add("fa-eye-slash");
      }
    }
  },
  mounted() {
    const el = document.body;
    el.classList.add("login-gradient");
  }
};
</script>

<style lang="less">
.login-gradient {
  background-repeat: no-repeat;
  background-size: cover;
  background-image: url("~@/assets/login-gradient-bg.svg");
  background-color: #f7f8f9;
}

.login-page .panel-default {
  border-color: #ddd;
}

.login-page .panel {
  margin-bottom: 20px;
  background-color: #fff;
  border: 1px solid transparent;
  border-radius: 4px;
  -webkit-box-shadow: 0 1px 1px rgba(0, 0, 0, 0.05);
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.05);
}

.login-page .panel-body {
  padding: 15px;
}

.login-page .panel-body h1,
.login-page .panel-body h2,
.login-page .panel-body h3,
.login-page .panel-body h4 {
  color: #202324;
}

.login-page .panel-body h2 {
  font-size: 22px;
  font-weight: bold;
  text-align: left;
}

.login-page p {
  font-size: 14px;
  color: #202324;
}

.login-page .panel-body h4 {
  margin-bottom: 1rem;
}

.login-page .panel:first-child {
  box-shadow: 0px 1px 30px #0000001a;
}

.login-page .panel-body h4 {
  font-size: 14px;
  font-weight: bold;
}

.login-page .panel-body {
  padding: 15px 15%;
}

.login-page .logo {
  margin: 1.5rem 0 2.5rem 0;
}

.login-page .panel .form-control {
  background-color: #f5f7f7;
  border: 1px solid #ebebeb;
  border-radius: 0.25rem;
  margin: 0.4rem 0;
}

.login-page .field-group {
  margin: 2rem 0;
}

.login-page .panel .form-control {
  font-size: 14px;
  color: #909192;
}

.login-page .panel .form-control::placeholder {
  font-size: 14px;
  color: #909192;
}

.login-page footer,
.login-page footer a,
.login-page footer a:hover {
  background-color: inherit;
  text-align: center;
  color: #202324;
  font-weight: bold;
  font-size: 12px;
}

.login-page footer a:hover {
  text-decoration: underline;
}

.login-page .btn-primary {
  background-color: #2c9cd6;
  border-radius: 0.25rem;
  border: none;
  font-size: 13px;
  font-weight: bold;
}

.login-page footer {
  .bi-heart-fill path {
    color: #ffdc00;
  }
}

.login-page .form-group small a {
  color: inherit;
}
</style>
