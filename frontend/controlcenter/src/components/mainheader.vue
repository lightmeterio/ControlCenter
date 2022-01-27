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

    <b-navbar toggleable="lg">
      <div class="container px-3 px-lg-0 d-flex justify-content-between">
        <router-link class="logo navbar-brand d-flex align-items-center" to="/">
          <img src="@/assets/logo-color-120.png" alt="Lightmeter logo" />
        </router-link>

        <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>

        <b-collapse id="nav-collapse" is-nav>
          <b-navbar-nav
            class="ml-auto d-flex justify-content-between align-items-center"
          >
            <div class="buttons">
              <router-link to="/">
                <translate>Observatory</translate>
              </router-link>

              <router-link
                v-on:click="trackClick('Detective', 'clickHeaderButton')"
                to="/detective"
              >
                <translate>Message Detective</translate>
              </router-link>

              <router-link
                v-on:click="trackClick('Settings', 'clickHeaderButton')"
                to="/settings"
              >
                <translate>Settings</translate>
              </router-link>

              <a v-b-modal.modal-about v-on:click="onGetApplicationInfo">
                <translate>About</translate>
              </a>

              <a :href="FeedbackMailtoLink" :title="FeedbackButtonTitle">
                <!-- prettier-ignore -->
                <translate>Feedback</translate>
              </a>
            </div>

            <div class="logout ml-3">
              <button v-on:click="onLogout">
                <translate>Log out</translate>
                <i class="fas fa-sign-out-alt"></i>
              </button>
            </div>
          </b-navbar-nav>
        </b-collapse>
      </div>
    </b-navbar>

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
        <strong> <translate>Version</translate>: </strong>
        {{ applicationData.version }}
        <br />
        <strong> <translate>Commit</translate>: </strong>
        {{ applicationData.commit }}
      </span>
      <br />
      <br />

      <p>
        <span class="link">
          <router-link to="/signals">
            <translate>View sent signals</translate>
          </router-link>
        </span>
        (
        <translate>telemetry data</translate>).
      </p>

      <strong>
        <translate>Get involved</translate>
      </strong>
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
            <translate>Newsletter</translate>
          </a>
        </li>
        <li>
          <a href="https://discuss.lightmeter.io/" target="_blank">Forum</a>
        </li>
        <li>
          <a href="https://twitter.com/lightmeterio" target="_blank">Twitter</a>
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
        <a href="https://www.gnu.org/licenses/agpl-3.0.en.html" target="_blank">
          <translate>AGPLv3 Licensed</translate>
        </a>
      </p>

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
  background: #e2f5fc 0% 0% no-repeat padding-box;
  opacity: 1;
  padding: 0;
}

header .logo img {
  height: 36px;
}

header .buttons {
  display: flex;
}

header .buttons a {
  margin-left: 0.75em;
}

header a {
  color: #111827;
  background-color: transparent;
  font-weight: 500;
  padding: 8px 12px;
  border-radius: 6px;
}

header a:hover {
  color: #111827;
  background-color: #b6e6f6;
  text-decoration: none;
}

header .logo.router-link-exact-active {
  background-color: transparent;
}

header .logo:hover {
  background-color: transparent;
}

header .router-link-exact-active {
  background-color: #b6e6f6;
}

header button {
  background: #fff;
  border: 1px solid #d1d5db;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  border-radius: 6px;
  color: #374151;
  padding: 10px 16px;
  font-weight: 500;
}

header button span {
  margin-right: 0.5em;
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

@media (max-width: 768px) {
  header .buttons {
    flex-direction: column;
  }
  header button span {
    margin-right: 0;
  }
  header .buttons a {
    margin-left: 0;
  }
}
</style>
