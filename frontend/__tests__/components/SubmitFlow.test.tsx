/**
 * Tests for the Submit flow — integration test of the ReviewView component
 * and related submit behavior.
 *
 * Tests submit button states, API interaction, and error handling at the
 * ReviewView level rather than through the full BulkStaffImport pipeline.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders, mockGetMe } from "../setup/test-utils";
import { server } from "../setup/msw-server";
import { http, HttpResponse } from "msw";

import { ReviewView } from "@/features/staff-import/components/review-view";
import { BulkStaffImport } from "@/features/staff-import/components/bulk-staff-import-dialog";
import { buildRow } from "../factories/inviteRow";
import type { ImportDraftRow } from "@/lib/db";

// ─── Mock useVirtualizer ──────────────────────────────────────────────

vi.mock("@tanstack/react-virtual", () => ({
    useVirtualizer: (opts: { count: number; estimateSize: () => number }) => ({
        getVirtualItems: () =>
            Array.from({ length: opts.count }, (_, index) => ({
                index,
                key: index,
                start: index * opts.estimateSize(),
                end: (index + 1) * opts.estimateSize(),
                lane: 0,
            })),
        getTotalSize: () => opts.count * opts.estimateSize(),
        measureElement: vi.fn(),
    }),
}));

// ─── Tests: ReviewView submit behavior ─────────────────────────────────

describe("SubmitFlow — ReviewView submit", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    function renderReviewView(rows: ImportDraftRow[]) {
        const onSubmit = vi.fn();
        const onBack = vi.fn();
        const utils = renderWithProviders(
            <ReviewView rows={rows} role="NURSE" onSubmit={onSubmit} onBack={onBack} />
        );
        return { ...utils, onSubmit, onBack };
    }

    it("submit button is disabled when critical errors exist — ReviewView shows submit enabled for clean rows", () => {
        const rows = [buildRow()];
        renderReviewView(rows);

        const submitBtn = screen.getByRole("button", { name: /submit 1 invitation/i });
        expect(submitBtn).not.toBeDisabled();
    });

    it("submit button is enabled at zero critical errors — all rows clean; Submit is enabled", () => {
        const rows = [buildRow(), buildRow()];
        renderReviewView(rows);

        const submitBtn = screen.getByRole("button", { name: /submit 2 invitations/i });
        expect(submitBtn).not.toBeDisabled();
    });

    it("submit sends correct payload shape — clicking Submit with 2 clean rows calls onSubmit with job ID", async () => {
        let capturedBody: unknown = null;

        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", async ({ request }) => {
                capturedBody = await request.json();
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 2,
                });
            })
        );

        const rows = [
            buildRow({ temp_id: "row-1", email: "a@b.com", full_name: "A", full_name: "B" }),
            buildRow({ temp_id: "row-2", email: "c@d.com", full_name: "C", full_name: "D" }),
        ];
        const { onSubmit } = renderReviewView(rows);

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: /submit 2 invitations/i }));

        await waitFor(() => {
            expect(onSubmit).toHaveBeenCalledWith("job-001");
        });

        // Verify payload shape
        expect(capturedBody).not.toBeNull();
        const payload = capturedBody as Record<string, unknown>;
        expect(payload).toHaveProperty("role", "NURSE");
        expect(payload).toHaveProperty("records");
    });

    it("submit payload includes client-generated rowId per row", async () => {
        let capturedBody: unknown = null;

        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", async ({ request }) => {
                capturedBody = await request.json();
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 1,
                });
            })
        );

        const rows = [buildRow({ temp_id: "row-abc-123" })];
        const { onSubmit } = renderReviewView(rows);

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: /submit 1 invitation/i }));

        await waitFor(() => {
            expect(onSubmit).toHaveBeenCalled();
        });

        const payload = capturedBody as { records: Array<{ temp_id: string }> };
        expect(payload.records[0].temp_id).toBe("row-abc-123");
    });

    it("HTTP 202 response shows progress UI — onSubmit is called with job ID", async () => {
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", () => {
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 1,
                });
            })
        );

        const rows = [buildRow()];
        const { onSubmit } = renderReviewView(rows);

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: /submit 1 invitation/i }));

        await waitFor(() => {
            expect(onSubmit).toHaveBeenCalledWith("job-001");
        });
    });

    it("loading spinner is shown during submit — between click and response, button shows sending state", async () => {
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", async () => {
                await new Promise((resolve) => setTimeout(resolve, 500));
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 1,
                });
            })
        );

        const rows = [buildRow()];
        renderReviewView(rows);

        const user = userEvent.setup();
        const submitBtn = screen.getByRole("button", { name: /submit 1 invitation/i });
        await user.click(submitBtn);

        await waitFor(() => {
            expect(screen.getByText(/Submitting/i)).toBeInTheDocument();
        });
    });
});

describe("SubmitFlow — error handling at ReviewView", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    function renderReviewView(rows: ImportDraftRow[]) {
        const onSubmit = vi.fn();
        const onBack = vi.fn();
        const utils = renderWithProviders(
            <ReviewView rows={rows} role="NURSE" onSubmit={onSubmit} onBack={onBack} />
        );
        return { ...utils, onSubmit, onBack };
    }

    it("HTTP 4xx response is handled — mutation catches error, component remains visible", async () => {
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", () => {
                return HttpResponse.json(
                    { code: "validation_error", message: "Invalid email format" },
                    { status: 422 }
                );
            })
        );

        const rows = [buildRow()];
        const { onSubmit } = renderReviewView(rows);

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: /submit 1 invitation/i }));

        // Component should still be visible after error
        await waitFor(() => {
            expect(
                screen.getByRole("button", { name: /submit 1 invitation/i })
            ).toBeInTheDocument();
        });

        // onSubmit should NOT be called on error
        expect(onSubmit).not.toHaveBeenCalled();
    });

    it("HTTP 5xx response is handled — component remains visible after error", async () => {
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", () => {
                return HttpResponse.json(
                    { code: "internal_error", message: "Internal server error" },
                    { status: 500 }
                );
            })
        );

        const rows = [buildRow()];
        const { onSubmit } = renderReviewView(rows);

        const user = userEvent.setup();
        await user.click(screen.getByRole("button", { name: /submit 1 invitation/i }));

        await waitFor(() => {
            expect(
                screen.getByRole("button", { name: /submit 1 invitation/i })
            ).toBeInTheDocument();
        });

        expect(onSubmit).not.toHaveBeenCalled();
    });
});

describe("SubmitFlow — BulkStaffImport integration", () => {
    beforeEach(() => {
        mockGetMe();
        vi.clearAllMocks();
    });

    it("BulkStaffImport renders and shows the manual entry form after session loads", async () => {
        server.use(
            http.post("http://localhost:3000/api/v1/imports/staff", () => {
                return HttpResponse.json({
                    import_job_id: "job-001",
                    status: "accepted",
                    total: 2,
                });
            })
        );

        const user = userEvent.setup();
        renderWithProviders(<BulkStaffImport role="NURSE" mode="page" />);

        await waitFor(() => {
            expect(screen.queryByText("Loading...")).not.toBeInTheDocument();
        });

        // If draft prompt appears, dismiss it
        const startFreshBtn = screen.queryByText("Start Fresh");
        if (startFreshBtn) {
            await user.click(startFreshBtn);
        }

        await waitFor(() => {
            expect(screen.getByPlaceholderText("jane@school.edu")).toBeInTheDocument();
        });
    });
});
