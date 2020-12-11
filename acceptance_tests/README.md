# Running user acceptance tests

These tests (also referred to as User Acceptance Tests) are found in the `acceptance_tests` directory and executed by [Gauge](https://gauge.org/) and [Taiko](https://github.com/getgauge/taiko). These tests are part of CI/CD and executed on every GitLab commit.

### Run tests locally

From the root directory, build controlcenter:

```bash
make release
cd acceptance_tests
# if you have chrome / chromium installed already, then disable duplicate chromium download...
# export TAIKO_SKIP_CHROMIUM_DOWNLOAD=1
# ... and set the path to your existing chrome / chromium binary
# export TAIKO_BROWSER_PATH=/usr/bin/chrome-gnome-shell
# get node dependencies including gauge and taiko
npm install
# set the path to necessary npm binaries
export PATH=$PATH:$PWD/node_modules/.bin
# execute tests (all tests, for convenience)
npm test
# execute gauge directly (for access to all gauge options)
npm run-script gauge run specs/
```

After doing all this you should see a Chrome / Chromium browser open, and tests start to run.
