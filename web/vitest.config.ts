import { defineConfig } from "vitest/config";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [tsconfigPaths()],
  test: {
    environment: "jsdom",
    setupFiles: ["./src/test/setup.ts"],
    globals: true,
    exclude: ["e2e/**", "node_modules/**"],
    pool: "forks",
    poolOptions: {
      forks: {
        maxForks: 2,
      },
    },
    testTimeout: 15000,
    teardownTimeout: 5000,
  },
});
