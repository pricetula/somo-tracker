/**
 * Global test setup for the Bulk Staff Invitation Utility tests.
 *
 * - Installs fake-indexeddb globals
 * - Configures MSW server
 * - Mocks ResizeObserver / IntersectionObserver
 * - Fixes Date.now()
 * - Sets env vars and session mocks
 * - Mocks EventSource for SSE
 */

import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, afterAll, beforeAll, beforeEach, vi } from "vitest";
import "fake-indexeddb/auto";

import { server } from "./msw-server";
import "./mock-event-source";

// ─── Fake IndexedDB ────────────────────────────────────────────────────

// fake-indexeddb/auto installs IDBFactory, IDBKeyRange, etc. globally.
// Clear IndexedDB between tests to prevent draft state leakage.
// NOTE: indexedDB.deleteDatabase returns an IDBOpenDBRequest (not a Promise),
// so we must wrap it in a proper Promise to actually wait for deletion.
function deleteDB(name: string): Promise<void> {
    return new Promise((resolve, reject) => {
        const req = indexedDB.deleteDatabase(name);
        req.onsuccess = () => resolve();
        req.onerror = () => reject(req.error);
        req.onblocked = () => resolve(); // Don't hang on blocked
    });
}

beforeEach(async () => {
    const dbs = await indexedDB.databases();
    await Promise.all(dbs.map((db) => deleteDB(db.name)));
});

// ─── MSW Server ────────────────────────────────────────────────────────

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// ─── DOM Environment Mocks ─────────────────────────────────────────────

// ResizeObserver is required by @tanstack/react-virtual
class MockResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
}
window.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver;

// IntersectionObserver
class MockIntersectionObserver {
    root: Element | null = null;
    rootMargin = "";
    thresholds: ReadonlyArray<number> = [];
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords(): IntersectionObserverEntry[] {
        return [];
    }
}
window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver;

// ─── Fixed Date ────────────────────────────────────────────────────────
// NOTE: Fake timers are NOT enabled globally because they break existing tests.
// Individual test files that need TTL testing should enable them with:
//   vi.useFakeTimers({ shouldAdvanceTime: true });
//   vi.setSystemTime(new Date("2025-01-15T10:00:00Z").getTime());

// ─── Environment Variables ─────────────────────────────────────────────

process.env.NEXT_PUBLIC_TENANT_ID = "tenant-abc";
process.env.NEXT_PUBLIC_API_URL = "http://localhost:3000";

// ─── Cleanup ──────────────────────────────────────────────────────────

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
});
