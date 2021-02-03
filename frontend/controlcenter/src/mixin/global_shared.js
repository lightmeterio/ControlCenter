// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { trackEvent, trackCLick, trackEventArray } from "@/lib/util";

export default {
  methods: {
    trackClick: trackCLick,
    trackEvent: trackEvent,
    trackEventArray: trackEventArray
  }
};
