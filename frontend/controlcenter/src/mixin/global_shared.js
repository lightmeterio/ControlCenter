// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

import { trackEvent, trackClick, trackEventArray } from "@/lib/util";

export default {
  methods: {
    trackClick: trackClick,
    trackEvent: trackEvent,
    trackEventArray: trackEventArray
  }
};
