/* globals gauge*/
"use strict";

// add Gauge functions here in order to use them, otherwise you'll get `ReferenceError: foobar is not defined`
const { alert, accept, click, openBrowser,write, closeBrowser, goto, press, screenshot, text, button, focus, textBox, toRightOf, toLeftOf, dropDown, waitFor, $ } = require('taiko');

const assert = require("assert");
const child_process = require("child_process")
const tmp = require("tmp")

tmp.setGracefulCleanup();

const headless = process.env.headless_chrome.toLowerCase() === 'true';

var lightmeterProcess = null

var tmpDir = tmp.dirSync()

beforeSuite(async() => {
    lightmeterProcess = child_process.execFile('../lightmeter', ['-workspace', tmpDir.name, '-stdin', '-listen', ':8080'])

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

gauge.screenshotFn = async function() {
    return await screenshot({ encoding: 'base64' });
};

step("Go to homepage", async () => {
    await goto('localhost:8080');
});

step("Go to registration page", async () => {
    await goto('localhost:8080/register');
});

step("Go to login page", async () => {
    await goto('localhost:8080/login');
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

step("Select option <option> from menu <menuName>", async (option, menuName) => {
    await dropDown(menuName).select(option)
});

step("Open datepicker menu", async () => {
    await waitFor(3000)
    await click($("div#time-interval-field > span"))
    await waitFor(4000)
    await click(text('Custom Range'))
});

step("Skip forward several months", async () => {
    var button = $("//html/body/div[2]/div[@class='drp-calendar left']/div[@class='calendar-table']/table[@class='table-condensed']/thead/tr[1]/th[1]")

    for (var i = 0; i < 12 ; i++) {
        await click(button)
    }
});

step("Set start date", async () => {
    await click($("//html/body/div[2]/div[@class='drp-calendar right']/div[@class='calendar-table']/table[@class='table-condensed']/tbody/tr[2]/td"))
});

step("Move forward some months", async () => {
    var button = $("//html/body/div[2]/div[@class='drp-calendar right']/div[@class='calendar-table']/table[@class='table-condensed']/thead/tr[1]/th[3]")

    for (var i = 0; i < 3; i++) {
        await click(button)
    }
});

step("Set end date", async () => {
    await click($("//html/body/div[2]/div[@class='drp-calendar right']/div[@class='calendar-table']/table[@class='table-condensed']/tbody/tr[2]/td"))
});

step("Click apply", async () => {
    await click(button("apply"))
});

step("Click logout", async () => {
    await click($("//*[contains(@class,'fa-sign-out-alt')]"))
});

step("Expect to see <pageText>", async (pageText) => {
    await text(pageText).exists()
});
