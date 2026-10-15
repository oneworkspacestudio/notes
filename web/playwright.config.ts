import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./tests",
  workers: 1,
  timeout: 45000,
  expect: { timeout: 10000 },
  use: {
    baseURL: "http://127.0.0.1:18895",
    launchOptions: {
      executablePath:
        process.env.NOTES_CHROME ||
        "C:/Program Files/Google/Chrome/Application/chrome.exe",
    },
    viewport: { width: 1440, height: 1000 },
  },
  reporter: [
    ["list"],
    ["json", { outputFile: "../test-results/browser.json" }],
  ],
  outputDir: "../test-results/playwright",
});
