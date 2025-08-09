import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL: process.env.WEB_URL || "http://127.0.0.1:3000",
    trace: "retain-on-failure",
    channel: process.env.PLAYWRIGHT_CHANNEL || undefined,
  },
});
