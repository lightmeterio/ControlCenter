/* globals gauge*/
"use strict";
const { alert, accept, click, openBrowser,write, closeBrowser, goto, press, screenshot, text, focus, textBox, toRightOf, toLeftOf, dropDown } = require('taiko');

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

step("Go to main page", async () => {
    await goto('localhost:8080');
});

step("Go to registration page", async () => {
    await goto('localhost:8080/register');
});

step("Focus on field with placeholder <placeholder>", async (placeholder) => {
    await focus(textBox({placeholder: placeholder}))
});

step("Click on <clickable>", async (clickable) => {
    await click(clickable)
});

step("Type <content>", async (content) => {
    await write(content)
});

step("Select <option> from menu <menuName>", async (option, menuName) => {
    await dropDown('Most of my mail is…').select('direct')
});
