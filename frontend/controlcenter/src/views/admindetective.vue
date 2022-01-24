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
        Check the delivery status of an email that was sent or received
      </p>

      <p class="mt-4" v-translate>
        You can use domains instead of email addresses (@domain.org,
        domain.org).<br />
        Leave sender or recipient blank, to view all emails to or from someone,
        or some domain.
      </p>

      <p class="mt-4" v-translate>
        You can enable a restricted version of the Message Detective for your
        end-users <router-link to="/settings">in the settings</router-link>.
      </p>

      <detective ref="detective"></detective>
    </b-container>

    <mainfooter></mainfooter>
  </div>
</template>

<script>
export default {
  name: "admindetective"
};
</script>
