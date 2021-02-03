<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<template>
  <header>
    <div class="navbar">
      <div class="container d-flex justify-content-between">
        <router-link class="logo navbar-brand d-flex align-items-center" to="/">
          <img src="@/assets/logo-color-120.png" alt="Lightmeter logo" />
        </router-link>
        <span
          v-on:click="trackClick('Settings', 'clickHeaderButton')"
          class="buttons"
        >
          <router-link to="/settings">
            <i
              class="fas fa-cog"
              data-toggle="tooltip"
              data-placement="bottom"
              :title="Settings"
            ></i
          ></router-link>
          <span v-b-modal.modal-about v-on:click="onGetApplicationInfo">
            <i
              class="fas fa-info-circle"
              data-toggle="tooltip"
              data-placement="bottom"
              :title="Information"
            ></i>
          </span>
          <span v-on:click="onLogout">
            <i
              class="fas fa-sign-out-alt"
              data-toggle="tooltip"
              data-placement="bottom"
              :title="LogOut"
            ></i>
          </span>
        </span>

        <b-modal
          ref="modal-about"
          id="modal-about"
          hide-footer
          :title="About"
          cancel-only
        >
          Lightmeter Control Center
          <br />
          <span id="release-info" v-if="applicationData">
            <strong><translate>Version</translate>:</strong> {{ applicationData.version }}
            <br />
            <strong><translate>Commit</translate>:</strong> {{ applicationData.commit }} <br />
            <strong><translate>Tag/branch</translate></strong>: {{ applicationData.tag_or_branch }}
          </span>

          <br />
          <br />
          <ul>
            <li>
              <a
                href="https://lightmeter.io/?pk_campaign=lmcc&pk_source=webui"
                target="_blank"
              >
                <!-- prettier-ignore -->
                <translate>Website</translate>
              </a>
            </li>
            <li>
              <a href="https://gitlab.com/lightmeter" target="_blank">GitLab</a>
            </li>
            <li>
              <a
                href="https://phplist.lightmeter.io/lists/?p=subscribe&id=1"
                target="_blank"
              >
                <!-- prettier-ignore -->
                <translate>Newsletter</translate></a
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
            >
              <!-- prettier-ignore -->
              <translate>Close</translate>
            </b-button>
          </div>
        </b-modal>
      </div>
    </div>
  </header>
</template>
<script>
import { getApplicationInfo, logout } from "../lib/api.js";

import tracking from "../mixin/global_shared.js";

export default {
  name: "mainheader",
  mixins: [tracking],
  data() {
    return {
      year: null,
      applicationData: null
    };
  },
  computed: {
    Home: function() {
      return this.$gettext("Home");
    },
    Settings: function() {
      return this.$gettext("Settings");
    },
    Information: function() {
      return this.$gettext("Information");
    },
    LogOut: function() {
      return this.$gettext("Log out");
    },
    About: function() {
      return this.$gettext("About");
    }
  },
  methods: {
    hideModal() {
      this.$refs["modal-about"].hide();
    },
    onGetApplicationInfo() {
      this.trackClick("About", "clickHeaderButton");
      let vue = this;
      getApplicationInfo().then(function(response) {
        vue.applicationData = response.data;
      });
    },
    onLogout() {
      this.trackClick("Logout", "clickHeaderButton");
      let vue = this;
      const redirect = () => {
        vue.$router.push({ name: "login" });
      };

      logout(redirect);
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

header .buttons span {
  margin-left: 0.25rem;
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
