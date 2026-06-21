import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// ─── Auto-cleanup ─────────────────────────────────────────────────────────

// Ensure React DOM is unmounted between tests to prevent state leakage.
afterEach(() => {
    cleanup();
});
