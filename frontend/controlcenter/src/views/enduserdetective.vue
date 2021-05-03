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
        <!-- prettier-ignore -->
        <translate>Search and detect messages</translate>
      </h2>

      <p class="mt-4">Lorem ipsum</p>

      <detective forEndUsers></detective>
    </b-container>

    <div
      style="width: 100%; background-color: #EFEFEF; display: flex; justify-content: center; align-items: center; margin-top: auto;"
    >
      <span>
        <!-- prettier-ignore -->
        <translate>Created with Lightmeter</translate>
      </span>
      <a
        href="https://lightmeter.io"
        class="logo navbar-brand d-flex align-items-center"
        style="margin-left: 1em;"
      >
        <img
          src="@/assets/logo-color-120.png"
          alt="Lightmeter logo"
          style="height: 24px;"
        />
      </a>
    </div>
  </div>
</template>
