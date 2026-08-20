import { defineConfig, devices } from "@playwright/test";

const PORT = 5173;
const BASE_URL = `http://localhost:${PORT}`;

export default defineConfig({
  testDir: ".",
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? "github" : "html",
  use: {
    baseURL: BASE_URL,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
  },
  projects: [
    {
      name: "default",
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "smoke",
      use: { ...devices["Desktop Chrome"] },
      testMatch: "**/*.smoke.spec.ts",
    },
    {
      name: "mobile",
      use: {
        ...devices["Pixel 7"],
      },
      testMatch: "**/*.mobile.spec.ts",
    },
  ],
  webServer: {
    command: `npm run dev -- --port ${PORT}`,
    port: PORT,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
});
