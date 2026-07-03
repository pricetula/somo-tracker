/**
 * sse.test.ts — tests for the useImportJobStatus hook (SSE progress).
 *
 * Uses the MockEventSource from __tests__/setup/mock-event-source.ts
 * which replaces global EventSource with a controllable mock.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useImportJobStatus } from "../hooks/use-import-job-status";
import { MockEventSource } from "../../../../__tests__/setup/mock-event-source";

// ─── Reset between tests ──────────────────────────────────────────────────

beforeEach(() => {
    MockEventSource.reset();
});

afterEach(() => {
    MockEventSource.reset();
});

// ─── Tests ─────────────────────────────────────────────────────────────────

describe("useImportJobStatus (SSE)", () => {
    it("returns initial pending state when jobId is provided", () => {
        const { result } = renderHook(() => useImportJobStatus("job-1"));
        expect(result.current.status).toBe("pending");
        expect(result.current.processedRecords).toBe(0);
    });

    it("returns initial state when jobId is null", () => {
        const { result } = renderHook(() => useImportJobStatus(null));
        expect(result.current.status).toBe("pending");
    });

    // Test 30: SSE initial event reflects current state
    it("TC30 — first 'state' event updates progress immediately, not from zero", async () => {
        const { result } = renderHook(() => useImportJobStatus("job-1"));

        // Simulate the initial state event (resume semantics)
        const instance = MockEventSource.instances[0];
        expect(instance).toBeDefined();

        // Use emit() which dispatches to addEventListener listeners
        instance.emit("state", {
            status: "processing",
            total_records: 200,
            processed_records: 75,
            success_count: 70,
            failed_count: 5,
            total_chunks: 4,
            processed_chunks: 2,
        });

        await waitFor(() => {
            expect(result.current.status).toBe("processing");
            expect(result.current.processedRecords).toBe(75);
            expect(result.current.totalRecords).toBe(200);
        });
    });

    // Test 38: EventSource network-blip reconnect is not treated as terminal failure
    it("TC38 — transient error event does not trigger terminal callback", async () => {
        const onTerminal = vi.fn();
        renderHook(() => useImportJobStatus("job-1", onTerminal));

        const instance = MockEventSource.instances[0];
        instance.triggerError();

        // onTerminal should NOT have been called (it's not a status: failed event)
        await vi.waitFor(() => {
            expect(onTerminal).not.toHaveBeenCalled();
        });
    });

    // Test 37: EventSource cleanup on unmount
    it("TC37 — closes EventSource when component unmounts", async () => {
        const { unmount } = renderHook(() => useImportJobStatus("job-1"));

        const instance = MockEventSource.instances[0];
        expect(instance.close).not.toHaveBeenCalled();

        unmount();

        expect(instance.close).toHaveBeenCalledTimes(1);
    });

    // Full success flow: completed status fires terminal callback
    it("fires terminal callback on 'completed' status", async () => {
        const onTerminal = vi.fn();
        const { result } = renderHook(() => useImportJobStatus("job-1", onTerminal));

        const instance = MockEventSource.instances[0];
        instance.emit("state", {
            job_id: "job-1",
            status: "completed",
            total_records: 100,
            processed_records: 100,
            success_count: 100,
            failed_count: 0,
        });

        await waitFor(() => {
            expect(result.current.status).toBe("completed");
        });

        // Allow terminal callback to be fired (deferred via setTimeout)
        await vi.waitFor(() => {
            expect(onTerminal).toHaveBeenCalledTimes(1);
        });
    });

    // completed_with_errors fires terminal callback
    it("fires terminal callback on 'completed_with_errors' status", async () => {
        const onTerminal = vi.fn();
        renderHook(() => useImportJobStatus("job-1", onTerminal));

        const instance = MockEventSource.instances[0];
        instance.emit("state", {
            job_id: "job-1",
            status: "completed_with_errors",
            total_records: 100,
            processed_records: 100,
            success_count: 95,
            failed_count: 5,
        });

        await vi.waitFor(() => {
            expect(onTerminal).toHaveBeenCalled();
        });
    });

    // job-level 'failed' status fires terminal callback
    it("fires terminal callback on 'failed' status", async () => {
        const onTerminal = vi.fn();
        renderHook(() => useImportJobStatus("job-1", onTerminal));

        const instance = MockEventSource.instances[0];
        instance.emit("state", {
            job_id: "job-1",
            status: "failed",
            total_records: 100,
            processed_records: 50,
            success_count: 30,
            failed_count: 20,
        });

        await vi.waitFor(() => {
            expect(onTerminal).toHaveBeenCalled();
        });
    });

    // Progress event updates state incrementally
    it("progress event updates state from SSE 'progress' type event", async () => {
        const { result } = renderHook(() => useImportJobStatus("job-1"));

        const instance = MockEventSource.instances[0];

        // Dispatch a progress-like event via emit
        instance.emit("progress", {
            job_id: "job-1",
            status: "processing",
            total_records: 200,
            processed_records: 50,
            success_count: 45,
            failed_count: 5,
            total_chunks: 4,
            processed_chunks: 1,
        });

        await waitFor(() => {
            expect(result.current.processedRecords).toBe(50);
            expect(result.current.processedChunks).toBe(1);
        });
    });
});
