/**
 * Tests for the ResultsSummary component.
 *
 * Covers all three result states: success, partial (207 Multi-Status),
 * and error (full failure). Verifies retry and start-new callbacks.
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/results-summary.test.tsx
 */

import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithQuery } from "@/__tests__/test-utils";
import { ResultsSummary } from "../components/results-summary";
import type { ImportResultSummary } from "../types";

// ─── Helpers ──────────────────────────────────────────────────────────────

function successSummary(overrides?: Partial<ImportResultSummary>): ImportResultSummary {
    return {
        total: 10,
        successCount: 10,
        failureCount: 0,
        failures: [],
        status: "success",
        ...overrides,
    };
}

function partialSummary(overrides?: Partial<ImportResultSummary>): ImportResultSummary {
    return {
        total: 10,
        successCount: 7,
        failureCount: 3,
        failures: [
            {
                index: 0,
                status: "error",
                full_name: "Failed Student",
                error_message: "Email already exists",
                field_errors: { email: "Duplicate email address" },
            },
            {
                index: 1,
                status: "error",
                full_name: "Another Failure",
                error_message: "Invalid phone number",
            },
            {
                index: 2,
                status: "error",
                full_name: "Third Failure",
                error_message: "Missing required field",
            },
        ],
        status: "partial",
        ...overrides,
    };
}

function errorSummary(overrides?: Partial<ImportResultSummary>): ImportResultSummary {
    return {
        total: 10,
        successCount: 0,
        failureCount: 10,
        failures: [],
        status: "error",
        message: "Server unavailable. Please try again.",
        ...overrides,
    };
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("ResultsSummary — Success", () => {
    it("renders success confirmation with count", () => {
        renderWithQuery(
            <ResultsSummary summary={successSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByText(/import complete/i)).toBeInTheDocument();
        expect(screen.getByText(/successfully imported/i)).toBeInTheDocument();
        expect(screen.getByText(/10/)).toBeInTheDocument();
    });

    it("uses singular 'student' when count is 1", () => {
        renderWithQuery(
            <ResultsSummary
                summary={successSummary({ successCount: 1, total: 1 })}
                onRetry={vi.fn()}
                onStartNew={vi.fn()}
            />
        );

        // Text is split across <span> elements, use function matcher
        expect(
            screen.getByText((_content, element) => {
                return (
                    element?.tagName === "P" &&
                    element.textContent?.includes("1") &&
                    element.textContent?.includes("student")
                );
            })
        ).toBeInTheDocument();
    });

    it("renders 'Start New Import' button in success state", () => {
        renderWithQuery(
            <ResultsSummary summary={successSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByRole("button", { name: /start new import/i })).toBeInTheDocument();
    });

    it("calls onStartNew when clicked in success state", async () => {
        const onStartNew = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ResultsSummary summary={successSummary()} onRetry={vi.fn()} onStartNew={onStartNew} />
        );

        await user.click(screen.getByRole("button", { name: /start new import/i }));
        expect(onStartNew).toHaveBeenCalledTimes(1);
    });
});

describe("ResultsSummary — Partial Success (207)", () => {
    it("renders partial success header with counts", () => {
        renderWithQuery(
            <ResultsSummary summary={partialSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByText(/partial success/i)).toBeInTheDocument();
        expect(screen.getByText(/7 students imported/i)).toBeInTheDocument();
        expect(screen.getByText(/3 failed/i)).toBeInTheDocument();
    });

    it("lists failed records with error messages", () => {
        renderWithQuery(
            <ResultsSummary summary={partialSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByText(/failed student/i)).toBeInTheDocument();
        expect(screen.getByText(/email already exists/i)).toBeInTheDocument();
        expect(screen.getByText(/another failure/i)).toBeInTheDocument();
        expect(screen.getByText(/invalid phone number/i)).toBeInTheDocument();
    });

    it("displays field-level errors for failed records", () => {
        renderWithQuery(
            <ResultsSummary summary={partialSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByText(/duplicate email address/i)).toBeInTheDocument();
    });

    it("renders both 'Retry Failed' and 'Start New Import' buttons", () => {
        renderWithQuery(
            <ResultsSummary summary={partialSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByRole("button", { name: /retry failed/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /start new import/i })).toBeInTheDocument();
    });

    it("calls onRetry when Retry Failed is clicked", async () => {
        const onRetry = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ResultsSummary summary={partialSummary()} onRetry={onRetry} onStartNew={vi.fn()} />
        );

        await user.click(screen.getByRole("button", { name: /retry failed/i }));
        expect(onRetry).toHaveBeenCalledTimes(1);
    });
});

describe("ResultsSummary — Error (Full Failure)", () => {
    it("renders failure header with message", () => {
        renderWithQuery(
            <ResultsSummary summary={errorSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByText(/import failed/i)).toBeInTheDocument();
        expect(screen.getByText(/server unavailable/i)).toBeInTheDocument();
    });

    it("falls back to default message when no message is provided", () => {
        renderWithQuery(
            <ResultsSummary
                summary={errorSummary({ message: undefined })}
                onRetry={vi.fn()}
                onStartNew={vi.fn()}
            />
        );

        expect(screen.getByText(/unexpected error occurred/i)).toBeInTheDocument();
    });

    it("renders both Retry and Start New buttons in error state", () => {
        renderWithQuery(
            <ResultsSummary summary={errorSummary()} onRetry={vi.fn()} onStartNew={vi.fn()} />
        );

        expect(screen.getByRole("button", { name: /^retry$/i })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /start new import/i })).toBeInTheDocument();
    });

    it("calls onRetry when Retry is clicked in error state", async () => {
        const onRetry = vi.fn();
        const user = userEvent.setup();

        renderWithQuery(
            <ResultsSummary summary={errorSummary()} onRetry={onRetry} onStartNew={vi.fn()} />
        );

        await user.click(screen.getByRole("button", { name: /^retry$/i }));
        expect(onRetry).toHaveBeenCalledTimes(1);
    });
});
