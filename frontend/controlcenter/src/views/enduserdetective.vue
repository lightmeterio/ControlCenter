<!--
SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>

SPDX-License-Identifier: AGPL-3.0-only
-->

<template>
  <div id="end-users-detective-page" class="d-flex flex-column min-vh-100">
    
    <b-container id="detective" class="main-content">
      <h2 class="form-heading">
        <!-- prettier-ignore -->
        <translate>Search for emails</translate>
      </h2>

      <detective forEndUsers=true></detective>
      
      <div class="progress-indicator-area" v-show="shouldShowProgressIndicator">
        <import-progress-indicator :label="importingLogs" @finished="handleProgressFinished"></import-progress-indicator>
      </div>

    </b-container>
  </div>
</template>
