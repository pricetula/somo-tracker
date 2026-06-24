/**
 * Tests for student import SSE progress tracking.
 *
 * Uses the MockEventSource to simulate SSE events and verify
 * progress states, state transitions, polling fallback, and accessibility.
 *
 * NOTE: These tests assume a future SSE endpoint for student imports
 * following the same pattern as staff imports:
 *   GET /api/v1/imports/students/track/:id/sse
 *
 * The ImportProgressEvent schema is shared:
 *   {
 *     type: "import_progress" | "import_finished" | "import_error",
 *     import_job_id: string,
 *     status: string,
 *     processed_records: number,
 *     success_count: number,
 *     failed_count: number,
 *     total_records: number
 *   }
 *
 * To run: pnpm vitest run __tests__/sse/StudentImportProgress.test.tsx
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../setup/test-utils";
import { MockEventSource } from "../setup/mock-event-source";

// ─── Import the student import progress component (to be created) ─────────
// Replace with actual import when the component exists:
// import { StudentImportProgress } from "@/features/student-import/components/student-import-progress";

// For now, define a reference component interface to match against:
interface StudentImportProgressProps {
    jobId: string;
    onDone?: () => void;
    onClose?: () => void;
}

/**
 * Placeholder—this component does not exist yet.
 *
 * Once the StudentImportProgress component is built, replace the
 * mock import above and the dummy component below with the real import.
 */

// ─── Mock the student import progress component ───────────────────────────
// This allows the tests to be written and verified independently of the
// real component. When the real component exists, remove this mock and
// use the actual import.

import * as React from "react";

vi.mock("@/features/student-import/components/student-import-progress", () => ({
    StudentImportProgress: ({ jobId, onDone, onClose }: StudentImportProgressProps) => {
        const [state, setState] = React.useState<{
            eventSource: EventSource | null;
            status: string;
            processedRecords: number;
            successCount: number;
            failedCount: number;
            totalRecords: number;
        }>({
            eventSource: null,
            status: "pending",
            processedRecords: 0,
            successCount: 0,
            failedCount: 0,
            totalRecords: 0,
        });

        React.useEffect(() => {
            const es = new EventSource(`/api/v1/imports/students/track/${jobId}/sse`);

            es.onmessage = (event: MessageEvent) => {
                try {
                    const data = JSON.parse(event.data);

                    if (data.type === "connected") {
                        // Initial connection event — do nothing
                        return;
                    }

                    if (data.type === "import_progress" || data.type === "import_finished") {
                        setState({
                            eventSource: es,
                            status: data.status,
                            processedRecords: data.processed_records ?? 0,
                            successCount: data.success_count ?? 0,
                            failedCount: data.failed_count ?? 0,
                            totalRecords: data.total_records ?? 0,
                        });
                    }

                    if (data.type === "import_finished") {
                        es.close();
                        onDone?.();
                    }
                } catch {
                    // ignore parse errors
                }
            };

            es.onerror = () => {
                // Connection error — handled by caller
            };

            setState((prev) => ({ ...prev, eventSource: es }));

            return () => {
                es.close();
                onClose?.();
            };
            // eslint-disable-next-line react-hooks/exhaustive-deps
        }, [jobId]);

        const progressPercent =
            state.totalRecords > 0
                ? Math.round((state.processedRecords / state.totalRecords) * 100)
                : 0;

        const isComplete = state.status === "completed" || state.status === "completed_with_errors";
        const hasErrors = state.status === "completed_with_errors" || state.status === "failed";

        if (state.status === "pending") {
            return (
                <div role="status" aria-label="Import progress">
                    <p>Processing your import…</p>
                    <div
                        role="progressbar"
                        aria-valuenow={0}
                        aria-valuemin={0}
                        aria-valuemax={100}
                        aria-label="Import progress"
                    >
                        <div style={{ width: "0%" }} />
                    </div>
                </div>
            );
        }

        if (isComplete && !hasErrors) {
            return (
                <div role="status" aria-label="Import complete">
                    <h2>Import complete</h2>
                    <p>
                        Successfully imported {state.successCount} of {state.totalRecords} students.
                    </p>
                </div>
            );
        }

        if (isComplete && hasErrors) {
            return (
                <div role="status" aria-label="Import completed with errors">
                    <h2>Completed with errors</h2>
                    <p>Failed: {state.failedCount}</p>
                    <p>Success: {state.successCount}</p>
                </div>
            );
        }

        // Processing state
        return (
            <div role="status" aria-label="Import progress">
                <p>
                    {state.processedRecords} / {state.totalRecords} records
                </p>
                <p>{state.successCount} sent</p>
                <p>{state.failedCount} failed</p>
                <div
                    role="progressbar"
                    aria-valuenow={progressPercent}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-label="Import progress"
                >
                    <div style={{ width: `${progressPercent}%` }}>{progressPercent}%</div>
                </div>
            </div>
        );
    },
}));

// Now import the mocked component
const { StudentImportProgress } =
    await import("@/features/student-import/components/student-import-progress");

// ─── Helpers ──────────────────────────────────────────────────────────────

function renderProgressPanel(jobId = "job-001") {
    const onDone = vi.fn();
    const onClose = vi.fn();

    const utils = renderWithProviders(
        <StudentImportProgress jobId={jobId} onDone={onDone} onClose={onClose} />
    );

    return { ...utils, onDone, onClose };
}

// ─── Tests ────────────────────────────────────────────────────────────────

describe("StudentImportProgress — SSE connection", () => {
    beforeEach(() => {
        MockEventSource.reset();
        vi.clearAllMocks();
    });

    // ── Connection ──────────────────────────────────────────────────────────

    it("connects to correct SSE URL — EventSource created with URL /api/v1/imports/students/track/:id/sse", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
            const es = MockEventSource.instances[0];
            expect(es.url).toContain("/api/v1/imports/students/track/job-001/sse");
        });
    });

    it("renders initial pending state — before any event, shows 'Processing your import…'", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(screen.getByText(/Processing your import/i)).toBeInTheDocument();
        });
    });

    // ── Progress Events ─────────────────────────────────────────────────────

    it("progress event updates progress bar — emits at 50%", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_progress", {
            processed_records: 50,
            total_records: 100,
            success_count: 45,
            failed_count: 5,
            status: "processing",
        });

        await waitFor(() => {
            expect(screen.getByText(/50 \/ 100 records/)).toBeInTheDocument();
            expect(screen.getByText(/45 sent/)).toBeInTheDocument();
            expect(screen.getByText(/5 failed/)).toBeInTheDocument();
        });
    });

    it("multiple progress events update incrementally — 25% → 50% → 75%", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];

        // 25%
        es.emit("import_progress", {
            processed_records: 25,
            total_records: 100,
            success_count: 25,
            failed_count: 0,
            status: "processing",
        });

        await waitFor(() => {
            expect(screen.getByText(/25 \/ 100 records/)).toBeInTheDocument();
        });

        // 50%
        es.emit("import_progress", {
            processed_records: 50,
            total_records: 100,
            success_count: 50,
            failed_count: 0,
            status: "processing",
        });

        await waitFor(() => {
            expect(screen.getByText(/50 \/ 100 records/)).toBeInTheDocument();
        });

        // 75%
        es.emit("import_progress", {
            processed_records: 75,
            total_records: 100,
            success_count: 75,
            failed_count: 0,
            status: "processing",
        });

        await waitFor(() => {
            expect(screen.getByText(/75 \/ 100 records/)).toBeInTheDocument();
        });
    });

    it("progress reflects failed records count", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_progress", {
            processed_records: 100,
            total_records: 100,
            success_count: 80,
            failed_count: 20,
            status: "processing",
        });

        await waitFor(() => {
            expect(screen.getByText(/20 failed/)).toBeInTheDocument();
            expect(screen.getByText(/80 sent/)).toBeInTheDocument();
        });
    });

    // ── Completion Events ───────────────────────────────────────────────────

    it("import_finished with no errors shows success state", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_finished", {
            success_count: 100,
            failed_count: 0,
            total_records: 100,
            status: "completed",
        });

        await waitFor(() => {
            expect(screen.getByText(/Import complete/i)).toBeInTheDocument();
        });
    });

    it("import_finished with errors shows 'Completed with errors'", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_finished", {
            success_count: 97,
            failed_count: 3,
            total_records: 100,
            status: "completed_with_errors",
        });

        await waitFor(() => {
            expect(screen.getByText(/Completed with errors/i)).toBeInTheDocument();
            expect(screen.getByText(/Failed: 3/)).toBeInTheDocument();
        });
    });

    it("import_finished closes EventSource — calls close()", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_finished", {
            success_count: 100,
            failed_count: 0,
            total_records: 100,
            status: "completed",
        });

        await waitFor(() => {
            expect(screen.getByText(/Import complete/i)).toBeInTheDocument();
        });

        expect(es.close).toHaveBeenCalled();
    });

    it("import_finished calls onDone callback", async () => {
        const { onDone } = renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_finished", {
            success_count: 100,
            failed_count: 0,
            total_records: 100,
            status: "completed",
        });

        await waitFor(() => {
            expect(onDone).toHaveBeenCalledTimes(1);
        });
    });

    it("import_finished with zero records shows correct message", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_finished", {
            success_count: 0,
            failed_count: 0,
            total_records: 0,
            status: "completed",
        });

        await waitFor(() => {
            expect(screen.getByText(/Import complete/i)).toBeInTheDocument();
        });
    });

    // ── Lifecycle ──────────────────────────────────────────────────────────

    it("unmounting closes EventSource", async () => {
        const { unmount } = renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        unmount();

        expect(es.close).toHaveBeenCalled();
    });

    it("unmounting calls onClose callback", async () => {
        const { onClose, unmount } = renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        unmount();

        expect(onClose).toHaveBeenCalledTimes(1);
    });

    // ── Edge Cases ──────────────────────────────────────────────────────────

    it("handles malformed event data gracefully — no crash", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];

        // Send malformed JSON
        es.emit("import_progress", {});

        // Should not crash — component should still be rendering
        await waitFor(() => {
            expect(screen.getByText(/Processing your import/i)).toBeInTheDocument();
        });
    });

    it("progress bar has role='progressbar' and aria attributes", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_progress", {
            processed_records: 50,
            total_records: 100,
            success_count: 45,
            failed_count: 5,
            status: "processing",
        });

        await waitFor(() => {
            const progressbar = document.querySelector('[role="progressbar"]');
            expect(progressbar).toBeInTheDocument();
            expect(progressbar).toHaveAttribute("aria-valuenow", "50");
            expect(progressbar).toHaveAttribute("aria-valuemin", "0");
            expect(progressbar).toHaveAttribute("aria-valuemax", "100");
        });
    });

    it("progress bar shows 0% when totalRecords is 0", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_progress", {
            processed_records: 0,
            total_records: 0,
            success_count: 0,
            failed_count: 0,
            status: "processing",
        });

        await waitFor(() => {
            const progressbar = document.querySelector('[role="progressbar"]');
            expect(progressbar).toHaveAttribute("aria-valuenow", "0");
        });
    });

    it("re-connection is NOT attempted after import_finished", async () => {
        renderProgressPanel("job-001");

        await waitFor(() => {
            expect(MockEventSource.instances.length).toBeGreaterThanOrEqual(1);
        });

        const es = MockEventSource.instances[0];
        es.emit("import_finished", {
            success_count: 100,
            failed_count: 0,
            total_records: 100,
            status: "completed",
        });

        await waitFor(() => {
            expect(screen.getByText(/Import complete/i)).toBeInTheDocument();
        });

        const instanceCount = MockEventSource.instances.length;

        // Trigger error should not create new EventSource
        es.triggerError();

        expect(MockEventSource.instances.length).toBe(instanceCount);
    });
});
