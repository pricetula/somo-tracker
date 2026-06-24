/**
 * Tests for the useSessionRecovery hook.
 *
 * Covers: loading state, session found (prompt), no session (clear),
 * resume action, discard action.
 *
 * Uses fake-indexeddb for IndexedDB operations and mock timers.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/use-session-recovery.test.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useSessionRecovery } from "../hooks/use-session-recovery";
import { saveSession, clearSession } from "../lib/indexeddb";
import type { ImportSession } from "../types";

// ─── Helpers ──────────────────────────────────────────────────────────────

function createSampleSession(): ImportSession {
    return {
        sessionId: "session-uuid-123",
        createdAt: "2026-06-24T10:00:00.000Z",
        lastUpdatedAt: "2026-06-24T10:05:00.000Z",
        currentStep: "validation",
        totalRecords: 50,
        ingestionPattern: "csv",
        mappingConfig: {
            nameColumns: ["full_name"],
            genderColumn: "gender",
            dobColumn: "date_of_birth",
            upiColumn: "upi_number",
            knecColumn: "knec_assessment_number",
            parentColumns: ["parent_name"],
            classColumns: ["class_name"],
        },
    };
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("useSessionRecovery", () => {
    beforeEach(async () => {
        // Ensure clean IndexedDB state
        await clearSession();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("starts in loading state", () => {
        const { result } = renderHook(() => useSessionRecovery());
        expect(result.current.action).toBe("loading");
    });

    it("transitions to clear when no session exists in IndexedDB", async () => {
        const { result } = renderHook(() => useSessionRecovery());

        await waitFor(() => {
            expect(result.current.action).toBe("clear");
        });
        expect(result.current.session).toBeNull();
    });

    it("transitions to prompt when a stored session is found", async () => {
        const session = createSampleSession();
        await saveSession(session);

        const { result } = renderHook(() => useSessionRecovery());

        await waitFor(() => {
            expect(result.current.action).toBe("prompt");
        });
        expect(result.current.session).not.toBeNull();
        expect(result.current.session!.sessionId).toBe("session-uuid-123");
        expect(result.current.session!.currentStep).toBe("validation");
    });

    it("resume action transitions to clear state", async () => {
        const session = createSampleSession();
        await saveSession(session);

        const { result } = renderHook(() => useSessionRecovery());

        await waitFor(() => {
            expect(result.current.action).toBe("prompt");
        });

        result.current.resume();

        await waitFor(() => {
            expect(result.current.action).toBe("clear");
        });
    });

    it("discard action clears stored session and transitions to clear", async () => {
        const session = createSampleSession();
        await saveSession(session);

        const { result } = renderHook(() => useSessionRecovery());

        await waitFor(() => {
            expect(result.current.action).toBe("prompt");
        });

        await result.current.discard();

        expect(result.current.session).toBeNull();
        expect(result.current.action).toBe("clear");

        // Verify IndexedDB is actually cleared
        const { hasStoredSession } = await import("../lib/indexeddb");
        const exists = await hasStoredSession();
        expect(exists).toBe(false);
    });

    it("handles IndexedDB read error gracefully (falls to clear)", async () => {
        // Mock hasStoredSession to throw
        const indexeddb = await import("../lib/indexeddb");
        vi.spyOn(indexeddb, "hasStoredSession").mockRejectedValueOnce(
            new Error("IndexedDB unavailable")
        );

        const { result } = renderHook(() => useSessionRecovery());

        await waitFor(() => {
            expect(result.current.action).toBe("clear");
        });
        expect(result.current.session).toBeNull();
    });
});
