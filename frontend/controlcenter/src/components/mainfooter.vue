<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <footer class="mt-auto">
    <div class="container">
      <div class="row justify-content-between">
        <div class="col-md-10 mt-md-0 mt-3 align-left">
          <a
            href="https://lightmeter.io/about/"
            title="About Lightmeter"
            target="_blank"
            ><translate>Thank you for using Lightmeter</translate></a
          >. &copy; {{ year }}.
          <span class="link">
            <a
              href="https://lightmeter.io/privacy-policy/"
              title="Read policy"
              target="_blank"
              ><translate>Privacy Policy</translate></a
            ></span
          >
          <span class="link">
            <a
              :href="FeedbackMailtoLink"
              :title="FeedbackButtonTitle"
              v-on:click="trackClick('Feedback', 'clickMailTo')"
              target="_blank"
              ><translate>Feedback</translate></a
            ></span
          >
          <b-button class="link" @click="runWalkthrough()"
            ><translate>Walkthrough</translate></b-button
          >
          <span
            class="link"
            v-b-modal.modal-telemetry
            data-toggle="tooltip"
            data-placement="bottom"
            title="Network Intelligence"
          >
            Network Intelligence
          </span>
        </div>

        <div class="col-md-2 mb-md-0 mb-2 align-right">
          <language-switcher></language-switcher>
        </div>
      </div>

      <b-modal
        ref="modal-telemetry"
        id="modal-telemetry"
        hide-footer
        title="Network Intelligence"
        cancel-only
      >
        <p>
          The following <i>Network Intelligence reports</i> are regularly sent
          to a central Lightmeter server:
        </p>
        <ul>
          <li>
            <b>connectionstats</b>: collects the IP addresses and timestamps of
            incoming SMTP connections that try to authenticate on ports 587 or
            465. This data is obtained from the Postfix logs. It is used to
            identify and help prevent brute force attacks via the SMTP protocol
            based on automated threat sharing between Lightmeter users.
          </li>
          <li>
            <b>insights</b>: collects general statistics about recently
            generated Lightmeter insights. This data is used to improve the
            frequency of insights generation, e.g. avoiding too many
            notifications.
          </li>
          <li>
            <b>logslinecount</b>: reports general non-identifiable information
            about which kinds of logs Postfix generates, and whether they are
            currently supported by <i>Lightmeter ControlCenter</i>. This is used
            to identify and prioritise support for unsupporter log events.
          </li>
          <li>
            <b>mailactivity</b>: collects statistics about the number of sent,
            bounced, deferred, and received messages. This helps identify
            Lightmeter performance issues and benchmark network health.
          </li>
          <li>
            <b>topdomains</b>: reports the domains that Postfix is managing
            locally. This is used to verify that reports are authentic.
          </li>
        </ul>
      </b-modal>
    </div>
  </footer>
</template>
<script>
import tracking from "../mixin/global_shared.js";
import shared_texts from "../mixin/shared_texts.js";
import { mapActions } from "vuex";

export default {
  name: "mainfooter",
  mixins: [tracking, shared_texts],
  data() {
    return {
      year: null
    };
  },
  mounted() {
    this.year = new Date().getFullYear();
  },
  methods: {
    runWalkthrough() {
      this.setWalkthroughNeedsToRunAction(true);
    },
    ...mapActions(["setWalkthroughNeedsToRunAction"])
  }
};
</script>
<style>
footer {
  padding: 0.5rem 0;
}
footer .btn,
footer .btn-secondary {
  padding: 0;
  margin: 0;
  font-size: 1em;
  background-color: inherit;
  border: none;
}
footer .link {
  padding-left: 0.8em;
}
footer .link::before {
  content: "\2022\00a0"; /* add bullet and space */
}
/* override default bootstrap focus button style */
footer .btn-secondary:focus,
footer .btn-secondary.focus {
  color: #87c528;
  background-color: inherit;
  border-color: #545b62;
  box-shadow: none;
  border: 0;
}
footer .btn-secondary:hover {
  color: #fff;
  background: none;
  border: none;
  text-decoration: underline;
}
footer .btn {
  vertical-align: unset;
}
</style>
