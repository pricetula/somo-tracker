import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
    test: {
        environment: "jsdom",
        include: [
            "src/**/*.test.ts",
            "src/**/*.test.tsx",
            "src/**/*.spec.ts",
            "src/**/*.spec.tsx",
            "__tests__/**/*.test.ts",
            "__tests__/**/*.test.tsx",
        ],
        exclude: [
            // StudentImportProgress component does not exist yet
            "__tests__/sse/StudentImportProgress.test.tsx",
        ],
        setupFiles: ["./__tests__/setup/vitest.setup.ts"],
        css: true,
    },
    resolve: {
        alias: {
            "@": path.resolve(__dirname, "./src"),
        },
    },
});
