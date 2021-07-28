<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="alerts">
    <div
      v-for="(alert, index) in alerts"
      :key="index"
      :class="statusClass(alert)"
    >
      {{ alert.message }}
    </div>
  </div>
</template>

<script>
import { mapState } from "vuex";

export default {
  computed: {
    ...mapState(["alerts"])
  },
  methods: {
    statusClass(alert) {
      return "alert alert-" + alert.severity;
    }
  }
};
</script>

<style lang="less">
div.alerts {
  z-index: 10;
  position: fixed;
  bottom: 3rem;
  width: 100vw;
  font-size: 16px;

  .alert {
    max-width: 600px;
    padding: 0.5rem 1rem;
    margin: auto;
    & + .alert {
      margin-top: 0.5rem;
    }
  }
}
</style>
