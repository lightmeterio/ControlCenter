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
            Usage
          </div>
          <ul class="card-text" style="margin-left: 0;padding-left: inherit;">
            <li>
              You can use domains instead of email addresses (@domain.org,
              domain.org)
            </li>
            <li>
              Leave sender or recipient blank, to view all emails to or from
              someone, or some domain
            </li>
            <li>
              You can enable a restricted view of the Message Detective for
              Mailbox users
              <router-link to="/settings">in the settings</router-link>
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
export default {
  name: "admindetective"
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
