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
  // Single worker everywhere: the API rate-limits 60 req/min per IP, and
  // parallel tests with open boards amplify each other — every mutation
  // broadcasts an SSE event that refetches every open board, so a handful
  // of concurrent specs trips 429 on /auth/me and logs the test out.
  workers: 1,
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
      // *.mobile.spec.ts belongs to the "mobile" project (touch viewport) —
      // running it on Desktop Chrome too would just double it up.
      testIgnore: "**/*.mobile.spec.ts",
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
