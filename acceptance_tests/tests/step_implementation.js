// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

/* globals gauge*/
"use strict";

// add Gauge functions here in order to use them, otherwise you'll get `ReferenceError: foobar is not defined`
const {
  alert,
  accept,
  click,
  openBrowser,write,
  closeBrowser,
  goto,
  press,
  screenshot,
  text,
  button,
  focus,
  textBox,
  toRightOf,
  toLeftOf,
  dropDown,
  waitFor,
  $,
  reload,
  currentURL } = require('taiko');

const assert = require("assert");
const child_process = require("child_process")
const tmp = require("tmp")
const path = require("path")
const fs = require('fs');

tmp.setGracefulCleanup();

const headless = process.env.headless_chrome.toLowerCase() === 'true';

var lightmeterProcess = null

var workspaceDir = tmp.dirSync()

function getLogsWithSingleDateTodayMinus45Days(originalLogFile) {
  let logs = "";
  try {
    const data = fs.readFileSync(originalLogFile, 'utf8');
    const lines = data.split(/\r?\n/);

    let somePreviousMonth = new Date(new Date() - 45 *1000*3600*24);  // now - 45 days
    let prevDate = somePreviousMonth.toISOString().substring(0,11);

    lines.forEach((line) => {
      line = line.replace( new RegExp("^\\d{4}-\\d{2}-\\d{2}T"), prevDate)
      logs += line + "\n"
    });
  }
  catch (err) {
    console.error(err);
  }

  return logs;
}

beforeSuite(async () => {
  let callback = function(error, stdout, stderr) {
    if (error) {
      console.warn(stdout)
      console.error(stderr)
      throw error
    }
  }

  lightmeterProcess = child_process.execFile(
    '../lightmeter',
    ['-workspace', workspaceDir.name, '-stdin', '-log_format', 'rfc3339', '-listen', ':8080'],
    callback
  )

  let logs = getLogsWithSingleDateTodayMinus45Days('../test_files/postfix_logs/individual_files/25_authclean_cleanup.log');
  lightmeterProcess.stdin.write(logs)

  return new Promise((r) => setTimeout(r, 2000)).then(async () => {
      await openBrowser({ headless: headless, args: ["--no-sandbox", "--no-first-run"] })
  })
});

step("Expect registration to fail", async () => {
  alert("Please select an option for 'Most of my mail is' - see help for details", async ({message}) => {
    console.log("accepting popup with message: " + message)
    await accept()
  })
});

afterSuite(async () => {
    await closeBrowser().then(function() {
        lightmeterProcess.kill()
    });
});

// Replaced previous deprecated screenshotFn by customScreenshotWriter, see
// https://docs.gauge.org/writing-specifications.html?&language=javascript#taking-custom-screenshots
gauge.customScreenshotWriter = async function () {
    const screenshotFilePath = path.join(process.env['gauge_screenshots_dir'], `screenshot-${process.hrtime.bigint()}.png`);
    await screenshot({ path: screenshotFilePath });
    return path.basename(screenshotFilePath);
};

// Add 'waitForEvents' following this comment: https://github.com/getgauge/taiko/issues/393#issuecomment-467719663
let waitOpt = {waitForEvents:['loadEventFired']};

step("Go to homepage", async () => {
    await goto('http://localhost:8080/#/', waitOpt);
    await reload(waitOpt)
});

step("Go to registration page", async () => {
    await goto('http://localhost:8080/#/register', waitOpt);
    await reload(waitOpt)
});

step("Go to login page", async () => {
    await goto('http://localhost:8080/#/login', waitOpt);
    await reload(waitOpt)
});

step("Go to detective", async () => {
  await goto('http://localhost:8080/#/detective', waitOpt);
  await reload(waitOpt)
});

step("Focus on field with placeholder <placeholder>", async (placeholder) => {
    await focus(textBox({placeholder: placeholder}))
});

step("Click on <clickable>", async (clickable) => {
    await click(button(clickable))
});

step("Type <content>", async (content) => {
    await write(content)
});

// NOTE: not used any more, was used for mailKind dropdown
step("Select option <option> from menu <menuName>", async (option, menuName) => {
    // Ugly workaroudn due a bug on taiko: https://github.com/getgauge/taiko/issues/1729
    //await dropDown(menuName).select(option)
    await dropDown().select(option)
});

step("Sleep <time> ms", async (ms) => {
    await waitFor(ms)
});

step("Open datepicker menu", async () => {
    await waitFor(1000)
    await click($(".vue-daterange-picker"))
    await waitFor(1000)
    await click(text('Custom Range'))
});

step("Skip forward several months", async () => {
    var button = $(".daterangepicker * .left * .prev")

    for (var i = 0; i < 12 ; i++) {
        await click(button)
    }
});

step("Set start date", async () => {
    // choose any time in the weekend
    await click($(".daterangepicker * .left * td.weekend"))
});

step("Move forward some months", async () => {
    var button = $(".daterangepicker * .left * .next")

    for (var i = 0; i < 3; i++) {
        await click(button)
    }
});

step("Set end date", async () => {
    await click($(".daterangepicker * .left * td.weekend"))
});

step("Datepicker last 3 months", async () => {
  await click($(".vue-daterange-picker"))
  await click(text('Last 3 months (all time)'))
});

step("Click logout", async () => {
    await waitFor(1000)
    await click($(".fa-sign-out-alt"))
});

step("Expect to see <pageText>", async (pageText) => {
    await assert.ok(await text(pageText).exists())
});

step("Expect to be in the main page", async () => {
    await waitFor(async() => {
      var url = await currentURL();
      console.log("current url:", url)
      return url == "http://localhost:8080/#/";
    })
});

step("Expect <x> detective results", async (x) => {
  assert.equal( (await $('.results .detective-result-cell').elements()).length, 1);
});
