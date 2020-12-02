import { trackEvent, trackCLick, trackEventArray } from "@/lib/util";

export default {
  methods: {
    trackClick: trackCLick,
    trackEvent: trackEvent,
    trackEventArray: trackEventArray
  }
};
