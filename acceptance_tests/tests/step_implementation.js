/* globals gauge*/
"use strict";
const { click, openBrowser,write, closeBrowser, goto, press, screenshot, text, focus, textBox, toRightOf } = require('taiko');
const assert = require("assert");
const headless = process.env.headless_chrome.toLowerCase() === 'true';

beforeSuite(async () => {
    await openBrowser({ headless: headless, args: ["--no-sandbox"] })
});

afterSuite(async () => {
    await closeBrowser();
});

gauge.screenshotFn = async function() {
    return await screenshot({ encoding: 'base64' });
};

step("Goto google", async () => {
    await goto('google.de');
});

step("Search for <query>", async (query) => {
    await write(query);
    await click('Google Suche')
});

step("Page contains <content>", async (content) => {
    assert.ok(await text(content).exists());
});
