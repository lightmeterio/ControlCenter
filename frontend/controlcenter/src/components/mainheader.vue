<template>
  <header>
    <div class="navbar">
      <div class="container d-flex justify-content-between">
        <a
          class="logo navbar-brand d-flex align-items-center"
          href="/"
          title="Home"
        >
          <img src="@/assets/logo-color-120.png" alt="Lightmeter logo" />
        </a>
        <span class="buttons">
          <router-link to="/settings">
            <i
              class="fas fa-cog"
              data-toggle="tooltip"
              data-placement="bottom"
              title="Settings"
            ></i
          ></router-link>

          <!--onclick="_paq.push(['trackEvent', 'Settings', 'clickHeaderButton']);" -->
          <span v-b-modal.modal-about v-on:click="onGetApplicationInfo">
            <i
              class="fas fa-info-circle"
              data-toggle="tooltip"
              data-placement="bottom"
              title="Information"
            ></i>
            <!--onclick="_paq.push(['trackEvent', 'About', 'clickHeaderButton']);" -->
          </span>
          <span v-on:click="onLogout">
            <i
              class="fas fa-sign-out-alt"
              data-toggle="tooltip"
              data-placement="bottom"
              title="Log out"
            ></i>
          </span>
          <!-- onclick="_paq.push(['trackEvent', 'Logout', 'clickHeaderButton']);" -->
        </span>

        <b-modal
          ref="modal-about"
          id="modal-about"
          hide-footer
          title="About"
          cancel-only
        >
          Lightmeter Control Center
          <!--{{translate "Lightmeter Control Center"}}-->
          <br />
          <span id="release-info" v-if="applicationData">
            <strong>Version:</strong> {{ applicationData.Version }}
            <br />
            <strong>Commit:</strong> {{ applicationData.Commit }} <br />
            <strong>Tag/branch</strong>: {{ applicationData.TagOrBranch }}
          </span>

          <br />
          <br />
          <ul>
            <li>
              <a
                href="https://lightmeter.io/?pk_campaign=lmcc&pk_source=webui"
                target="_blank"
                >Website
                <!-- {{translate "Website"}}--></a
              >
            </li>
            <li>
              <a href="https://gitlab.com/lightmeter" target="_blank">GitLab</a>
            </li>
            <li>
              <a
                href="https://phplist.lightmeter.io/lists/?p=subscribe&id=1"
                target="_blank"
                >Newsletter
                <!-- {{translate "Newsletter"}}--></a
              >
            </li>
            <li>
              <a href="https://twitter.com/lightmeterio" target="_blank"
                >Twitter</a
              >
            </li>
            <li>
              <a href="https://mastodon.social/@lightmeter/" target="_blank"
                >Mastodon</a
              >
            </li>
          </ul>
          <div class="custom-modal-footer modal-footer">
            <b-button
              class="btn-cancel"
              variant="outline-danger"
              @click="hideModal"
              >Close</b-button
            >
          </div>
        </b-modal>
      </div>
    </div>
  </header>
</template>
<script>
import { getApplicationInfo, logout } from "../lib/api.js";

export default {
  name: "mainheader",
  data() {
    return {
      year: null,
      applicationData: null
    };
  },
  methods: {
    hideModal() {
      this.$refs["modal-about"].hide();
    },
    onGetApplicationInfo() {
      let vue = this;
      getApplicationInfo().then(function(response) {
        vue.applicationData = response.data;
      });
    },
    onLogout() {
      logout();
    }
  }
};
</script>
<style>
header .navbar {
  height: 70px;
  background: #ffffff 0% 0% no-repeat padding-box;
  box-shadow: 0px 6px 8px #00000029;
  opacity: 1;
  padding: 0;
}

header .logo img {
  height: 36px;
}

header .buttons,
header a,
header a:hover {
  color: #2c9cd6;
}

header span.buttons {
  cursor: pointer;
}

header span.buttons svg {
  margin-left: 0.5rem;
}

#modal-about .btn-cancel {
  background: #ff5c6f33 0% 0% no-repeat padding-box;
  border: 1px solid #ff5c6f;
  border-radius: 2px;
  opacity: 0.8;
  text-align: center;
  font: normal normal bold 14px/24px Open Sans;
  letter-spacing: 0px;
  color: #820d1b;
}
</style>
