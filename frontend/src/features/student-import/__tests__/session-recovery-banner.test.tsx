/**
 * Tests for the SessionRecoveryBanner component.
 *
 * Verifies rendering of the timestamp message, resume/discard actions,
 * and interaction callbacks.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/session-recovery-banner.test.tsx
 */

import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithQuery } from "@/__tests__/test-utils";
import { SessionRecoveryBanner } from "../components/session-recovery-banner";
import type { ImportSession } from "../types";

// ─── Helpers ──────────────────────────────────────────────────────────────

function createSampleSession(overrides: Partial<ImportSession> = {}): ImportSession {
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
        ...overrides,
    };
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("SessionRecoveryBanner", () => {
    it("renders the session timestamp in the message", () => {
        const session = createSampleSession({
            createdAt: "2026-06-24T10:00:00.000Z",
        });
        renderWithQuery(
            <SessionRecoveryBanner session={session} onResume={vi.fn()} onDiscard={vi.fn()} />
        );

        expect(screen.getByText(/unfinished import session/i)).toBeInTheDocument();
        // The date is rendered via toLocaleString(), so we check for partial content
        expect(screen.getByText(/6\/24\/2026/i)).toBeInTheDocument();
    });

    it("renders both actions: Resume Session and Discard & Start New", () => {
        const session = createSampleSession();
        renderWithQuery(
            <SessionRecoveryBanner session={session} onResume={vi.fn()} onDiscard={vi.fn()} />
        );

        expect(screen.getByRole("button", { name: /resume session/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /discard.*start new/i })).toBeInTheDocument();
    });

    it("calls onResume when Resume Session is clicked", async () => {
        const onResume = vi.fn();
        const session = createSampleSession();
        const user = userEvent.setup();

        renderWithQuery(
            <SessionRecoveryBanner session={session} onResume={onResume} onDiscard={vi.fn()} />
        );

        await user.click(screen.getByRole("button", { name: /resume session/i }));
        expect(onResume).toHaveBeenCalledTimes(1);
    });

    it("calls onDiscard when Discard & Start New is clicked", async () => {
        const onDiscard = vi.fn();
        const session = createSampleSession();
        const user = userEvent.setup();

        renderWithQuery(
            <SessionRecoveryBanner session={session} onResume={vi.fn()} onDiscard={onDiscard} />
        );

        await user.click(screen.getByRole("button", { name: /discard.*start new/i }));
        expect(onDiscard).toHaveBeenCalledTimes(1);
    });

    it("displays a manual ingestion pattern session correctly", () => {
        const session = createSampleSession({ ingestionPattern: "manual" });
        renderWithQuery(
            <SessionRecoveryBanner session={session} onResume={vi.fn()} onDiscard={vi.fn()} />
        );

        expect(screen.getByText(/unfinished import session/i)).toBeInTheDocument();
    });

    it("displays a session with zero records", () => {
        const session = createSampleSession({ totalRecords: 0 });
        renderWithQuery(
            <SessionRecoveryBanner session={session} onResume={vi.fn()} onDiscard={vi.fn()} />
        );

        expect(screen.getByText(/unfinished import session/i)).toBeInTheDocument();
    });
});
