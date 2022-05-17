<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div
    id="admin-detective-page"
    class="d-flex flex-column min-vh-100 text-left"
  >
    <mainheader></mainheader>

    <b-container id="detective" class="main-content">
      <h2 class="form-heading mt-5">
        <translate>Message Detective</translate>
      </h2>

      <p class="mt-4" v-translate>
        Check the delivery status of an email that was sent or received
      </p>

      <div class="card border-info mb-12 lg-12 sm-12">
        <div class="card-body text-info">
          <div class="card-title">
            <i class="fa fa-info-circle"></i>
            <translate>Usage</translate>
          </div>
          <ul class="card-text" style="margin-left: 0;padding-left: inherit;">
            <li>
              <translate
                >You can use domains instead of email addresses (@domain.org,
                domain.org). Some common domains you can search for are:
              </translate>
              <ul>
                <li>
                  <strong>google.com</strong>:
                  <translate
                    >for GMail and other domains hosted by Google.</translate
                  >
                </li>
                <li>
                  <strong>outlook.com</strong>:
                  <translate
                    >for Outlook, Hotmail and other domains hosted by
                    Microsoft.</translate
                  >
                </li>
              </ul>
            </li>
            <li>
              <translate>
                Leave sender or recipient blank, to view all emails to or from
                someone, or some domain.
              </translate>
            </li>
            <li v-if="!simpleViewEnabled">
              You can enable a restricted view of the Message Detective for
              Mailbox users
              <router-link to="/settings">in the settings</router-link>.
            </li>
          </ul>
        </div>
      </div>

      <detective ref="detective"></detective>
    </b-container>

    <mainfooter></mainfooter>
  </div>
</template>

<script>
import { getSettings } from "../lib/api.js";

export default {
  name: "admindetective",
  data() {
    return {
      simpleViewEnabled: true
    };
  },
  mounted() {
    let vue = this;

    getSettings().then(function(response) {
      vue.simpleViewEnabled = response.data.feature_flags.enable_simple_view;
    });
  }
};
</script>

<style lang="less">
.text-info {
  color: #227aaf;
  border-color: #227aaf;
  .card-title {
    font-size: 110%;
  }
}
</style>
