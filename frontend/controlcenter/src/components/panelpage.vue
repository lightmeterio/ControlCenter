<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div class="panel-page d-flex flex-column min-vh-100">
    <b-container>
      <b-row class="justify-content-center vh-93">
        <b-col sm="5" class="align-self-center">
          <div class="panel panel-default mx-auto">
            <div class="panel-body">
              <b-row class="justify-content-center">
                <b-col lg="10">
                  <img
                    class="logo w-100"
                    src="@/assets/logo-color-120.svg"
                    alt="Lightmeter logo"
                  />
                </b-col>
              </b-row>
              <slot></slot>
            </div>
          </div>
        </b-col>
      </b-row>
    </b-container>
    <auth-page-footer></auth-page-footer>
  </div>
</template>
<script>
export default {
  name: "panel-page"
};
</script>
<style>
.panel-page .field-group {
  margin: 1rem 0;
}

.panel-page {
  margin-top: 1em;
}
</style>
