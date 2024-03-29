<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div
    id="end-users-detective-page"
    class="d-flex flex-column min-vh-100  text-left"
  >
    <header
      style="width: 100%; height: 1em; background: radial-gradient(ellipse at left top, #2a93d6, #3dd9d6);"
    ></header>

    <b-container id="detective" class="main-content">
      <h2 class="form-heading mt-5">
        <translate>Search and detect messages</translate>
      </h2>

      <p class="mt-4">
        <translate
          >Check the delivery status of an email that you sent or
          received</translate
        >
      </p>

      <detective forEndUsers ref="detective" @onResults="onResults"></detective>

      <b-container class="ml-auto p-2 detective-actions">
        <b-button
          :disabled="!canEscalateResults"
          v-show="hasAnyResults"
          variant="primary"
          @click="escalateMessage"
        >
          <translate>Escalate</translate>
        </b-button>
      </b-container>
    </b-container>

    <div class="detective-footer">
      <span>
        <translate>Created with Lightmeter</translate>
      </span>
      <a
        href="https://lightmeter.io"
        class="logo navbar-brand d-flex align-items-center"
        style="margin-left: 1em;"
      >
        <img
          src="@/assets/logo-color-120.svg"
          alt="Lightmeter logo"
          style="height: 24px;"
        />
      </a>
    </div>
  </div>
</template>

<script>
import axios from "axios";
axios.defaults.withCredentials = true;

import { escalateMessage } from "@/lib/api.js";
import tracking from "@/mixin/global_shared.js";

function resultCanBeEscalated(messages) {
  for (let entries of messages) {
    for (let entry of entries.entries) {
      if (entry.status != "sent") {
        return true;
      }
    }
  }

  return false;
}

export default {
  name: "searchmessage",
  mixins: [tracking],
  data() {
    return {
      canEscalateResults: false,
      escalationSender: "",
      escalationRecipient: "",
      escalationInterval: {},
      someID: "",
      hasAnyResults: false
    };
  },
  methods: {
    onResults(results, sender, recipient, interval, some_id) {
      this.escalationSender = sender;
      this.escalationRecipient = recipient;
      this.escalationInterval = interval;
      this.someID = some_id;
      this.canEscalateResults = resultCanBeEscalated(results.messages);
      this.hasAnyResults = results.messages.length > 0;
    },
    escalateMessage() {
      let vue = this;
      escalateMessage(
        this.escalationSender,
        this.escalationRecipient,
        this.escalationInterval.startDate,
        this.escalationInterval.endDate,
        this.someID
      ).then(function() {
        vue.canEscalateResults = false;
      });
    }
  }
};
</script>

<style lang="less" scoped>
.detective-footer {
  width: 100%;
  background-color: #efefef;
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: auto;
  height: 60px;
}

.detective-actions {
  margin-top: 20px;
}
</style>
