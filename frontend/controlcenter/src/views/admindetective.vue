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
        <!-- prettier-ignore -->
        <translate>Message Detective</translate>
      </h2>

      <p class="mt-4" v-translate>
        Check the delivery status of an email that you sent or received
      </p>

      <detective ref="detective"></detective>
    </b-container>

    <mainfooter></mainfooter>
  </div>
</template>

<script>
import auth from "@/mixin/auth.js";

export default {
  name: "admindetective",
  mixins: [auth],
  data() {
    return {
      neededAuth: true
    };
  }
};
</script>
