<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <header>
    <walkthrough
      :visible="walkthroughNeedsToRun"
      @finished="handleWalkthroughCompleted"
    ></walkthrough>
    <div class="navbar">
      <div class="container d-flex justify-content-between">
        <router-link class="logo navbar-brand d-flex align-items-center" to="/">
          <img src="@/assets/logo-color-120.png" alt="Lightmeter logo" />
        </router-link>
        <span class="buttons">
          <span v-on:click="trackClick('Detective', 'clickHeaderButton')">
            <router-link to="/detective">
              <i
                class="fas fa-search"
                data-toggle="tooltip"
                data-placement="bottom"
                :title="Detective"
              ></i
            ></router-link>
          </span>
          <span v-on:click="trackClick('Settings', 'clickHeaderButton')">
            <router-link to="/settings">
              <i
                class="fas fa-cog"
                data-toggle="tooltip"
                data-placement="bottom"
                :title="Settings"
              ></i
            ></router-link>
          </span>
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
          Lightmeter ControlCenter
          <br />
          <span id="release-info" v-if="applicationData">
            <strong><translate>Version</translate>:</strong>
            {{ applicationData.version }}
            <br />
            <strong><translate>Commit</translate>:</strong>
            {{ applicationData.commit }}
          </span>
          <br />
          <br />

          <p>
            <span class="link">
              <router-link to="/signals"
                ><translate>View sent signals</translate></router-link
              >
            </span>
            (<translate>telemetry data</translate>).
          </p>

          <strong><translate>Get involved</translate></strong>
          <br />
          <ul>
            <li>
              <a
                href="https://lightmeter.io/?pk_campaign=lmcc&pk_source=webui"
                target="_blank"
              >
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
                <translate>Newsletter</translate></a
              >
            </li>
            <li>
              <a href="https://discuss.lightmeter.io/" target="_blank">Forum</a>
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
            <li>
              <a href="https://t.me/lightmeterio" target="_blank">Telegram</a>
            </li>
          </ul>

          <p>
            <translate>Open Source software</translate>,
            <a
              href="https://www.gnu.org/licenses/agpl-3.0.en.html"
              target="_blank"
            >
              <translate>AGPLv3 Licensed</translate>
            </a>
          </p>

          <div class="custom-modal-footer modal-footer">
            <b-button
              class="btn-cancel"
              variant="outline-danger"
              @click="hideModal"
            >
              <translate>Close</translate>
            </b-button>
          </div>
        </b-modal>
      </div>
    </div>
  </header>
</template>
<script>
import { getApplicationInfo, logout, getSettings } from "../lib/api.js";
import tracking from "../mixin/global_shared.js";
import { mapActions, mapState } from "vuex";

export default {
  name: "mainheader",
  mixins: [tracking],
  mounted() {
    let vue = this;

    getSettings().then(function(response) {
      vue.setWalkthroughNeedsToRunAction(
        !response.data.walkthrough || !response.data.walkthrough.completed
      );
    });
  },
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
    Detective: function() {
      return this.$gettext("Mail Detective");
    },
    Information: function() {
      return this.$gettext("Information");
    },
    LogOut: function() {
      return this.$gettext("Log out");
    },
    About: function() {
      return this.$gettext("About");
    },
    ...mapState(["walkthroughNeedsToRun"])
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
    },
    handleWalkthroughCompleted() {
      this.setWalkthroughNeedsToRunAction(false);
    },
    ...mapActions(["setWalkthroughNeedsToRunAction"])
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
