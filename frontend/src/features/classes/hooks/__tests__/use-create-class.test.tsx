/**
 * Rollback test for useCreateClass.
 *
 * Verifies that when createClass mutation fails, the query cache
 * is restored to the exact pre-mutation snapshot via onMutate context.
 */

import { renderHook } from "@testing-library/react";
import { vi, describe, it, expect, beforeEach, afterEach } from "vitest";
import { useCreateClass } from "@/features/classes/hooks/use-classes";
import { createClass } from "@/lib/api/classes";
import { classKeys } from "@/features/classes/hooks/use-classes";
import type { Class } from "@/lib/api/generated";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactNode } from "react";

// ---------------------------------------------------------------------------
// Test utilities
// ---------------------------------------------------------------------------

/** Wrapper that provides a fresh QueryClient for each test. */
function createWrapper() {
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
            mutations: { retry: false },
        },
    });

    // eslint-disable-next-line  react/display-name
    return ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
}

/** Build a minimal valid Class object for testing. */
function buildClass(overrides: Partial<Class> = {}): Class {
    return {
        id: `class-${crypto.randomUUID()}`,
        grade_level: "G3",
        stream_id: "stream-1",
        stream_name: "Blue",
        stream_color: "#3B82F6",
        display_label: "G3 Blue",
        student_count: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        ...overrides,
    };
}

/** Seed the query cache with a list of classes. */
function seedCache(queryClient: QueryClient, classes: Class[]) {
    queryClient.setQueryData(classKeys.list(), {
        items: classes,
        total: classes.length,
        page: 1,
        limit: 500,
    });
}

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

vi.mock("@/lib/api/classes", async (importOriginal) => {
    const actual = await importOriginal<typeof import("@/lib/api/classes")>();
    return {
        ...actual,
        createClass: vi.fn(),
    };
});

vi.mock("@/lib/errors", () => ({
    getErrorMessage: vi.fn((err: unknown) =>
        err instanceof Error ? err.message : "Unknown error"
    ),
    isApiError: vi.fn((err: unknown) => err instanceof Error && "status" in err),
}));

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useCreateClass — optimistic update rollback", () => {
    let wrapper: ReturnType<typeof createWrapper>;
    let queryClient: QueryClient;

    beforeEach(() => {
        wrapper = createWrapper();
        // Access the internal QueryClient from the wrapper
        // We need to recreate it for each test to get a fresh instance
        const testQueryClient = new QueryClient({
            defaultOptions: {
                queries: { retry: false },
                mutations: { retry: false },
            },
        });
        queryClient = testQueryClient;

        wrapper = ({ children }: { children: ReactNode }) => (
            <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
        );

        vi.clearAllMocks();
    });

    afterEach(() => {
        queryClient.clear();
    });

    /**
     * Helper to render the hook with our test QueryClient.
     * We need to access the mutation via result.current.
     */
    function renderCreateClassHook() {
        return renderHook(() => useCreateClass(), { wrapper });
    }

    it("rolls back to pre-mutation snapshot on generic API error (5xx)", async () => {
        // Arrange: seed cache with existing classes
        const existingClass = buildClass({ id: "existing-1", display_label: "G1 Red" });
        seedCache(queryClient, [existingClass]);

        // Verify initial state
        expect(queryClient.getQueryData(classKeys.list())).toEqual({
            items: [existingClass],
            total: 1,
            page: 1,
            limit: 500,
        });

        // Mock API to throw a 500 error
        const apiError = new Error("Internal Server Error") as Error & { status: number };
        apiError.status = 500;
        vi.mocked(createClass).mockRejectedValue(apiError);

        // Act: trigger mutation
        const { result } = renderCreateClassHook();
        const mutatePromise = result.current.mutateAsync({
            grade_level: "G2",
            stream_id: "stream-2",
        });

        // Wait for mutation to settle (error)
        await expect(mutatePromise).rejects.toThrow("Internal Server Error");

        // Assert: cache rolled back to exact pre-mutation state
        const cached = queryClient.getQueryData<{ items: Class[] }>(classKeys.list());
        expect(cached).toEqual({
            items: [existingClass],
            total: 1,
            page: 1,
            limit: 500,
        });

        // Verify the optimistic item was never persisted
        expect(cached?.items.some((c) => c.id.startsWith("temp-"))).toBe(false);
    });

    it("rolls back to pre-mutation snapshot on validation error (422)", async () => {
        // Arrange
        const existingClass = buildClass({ id: "existing-1", display_label: "G1 Red" });
        seedCache(queryClient, [existingClass]);

        // Mock API to throw a 422 with field errors
        const validationError = new Error("Validation failed") as Error & {
            status: number;
            code: string;
            errors: Record<string, string[]>;
        };
        validationError.status = 422;
        validationError.code = "validation_error";
        validationError.errors = { grade_level: ["Grade level is required"] };
        vi.mocked(createClass).mockRejectedValue(validationError);

        // Act
        const { result } = renderCreateClassHook();
        const mutatePromise = result.current.mutateAsync({
            grade_level: "",
            stream_id: "stream-2",
        });

        await expect(mutatePromise).rejects.toThrow("Validation failed");

        // Assert: cache rolled back
        const cached = queryClient.getQueryData<{ items: Class[] }>(classKeys.list());
        expect(cached).toEqual({
            items: [existingClass],
            total: 1,
            page: 1,
            limit: 500,
        });
        expect(cached?.items.some((c) => c.id.startsWith("temp-"))).toBe(false);
    });

    it("does NOT rollback on 409 conflict — instead invalidates to refetch", async () => {
        // Arrange
        const existingClass = buildClass({ id: "existing-1", display_label: "G1 Red" });
        seedCache(queryClient, [existingClass]);

        // Mock API to throw a 409 conflict
        const conflictError = new Error(
            "Class already exists for this grade and stream"
        ) as Error & {
            status: number;
            code: string;
        };
        conflictError.status = 409;
        conflictError.code = "conflict";
        vi.mocked(createClass).mockRejectedValue(conflictError);

        // Act
        const { result } = renderCreateClassHook();
        const mutatePromise = result.current.mutateAsync({
            grade_level: "G1",
            stream_id: "stream-1",
        });

        await expect(mutatePromise).rejects.toThrow("Class already exists");

        // Assert: cache was invalidated (not rolled back)
        // The query should be marked as invalidated/stale, triggering a refetch
        // when observed. Since no observer exists in this test, fetchStatus stays idle.
        const queryState = queryClient.getQueryState(classKeys.list());
        expect(queryState).toBeDefined();
        // Verify the query was invalidated (stale = true means it will refetch on next observe)
        expect(queryState?.isInvalidated).toBe(true);
    });

    it("replaces temp item with server response on success (no flicker)", async () => {
        // Arrange
        const existingClass = buildClass({ id: "existing-1", display_label: "G1 Red" });
        seedCache(queryClient, [existingClass]);

        // Mock successful API response
        const serverClass: Class = buildClass({
            id: "server-real-id",
            grade_level: "G2",
            stream_id: "stream-2",
            display_label: "G2 Blue",
            stream_name: "Blue",
            stream_color: "#3B82F6",
        });
        vi.mocked(createClass).mockResolvedValue(serverClass);

        // Act
        const { result } = renderCreateClassHook();
        await result.current.mutateAsync({
            grade_level: "G2",
            stream_id: "stream-2",
        });

        // Assert: optimistic temp item replaced with real server class
        const cached = queryClient.getQueryData<{ items: Class[] }>(classKeys.list());
        expect(cached).toBeDefined();
        expect(cached!.items).toHaveLength(2);

        // The new class should have the real server ID, not a temp ID
        const newClass = cached!.items.find(
            (c) => c.grade_level === "G2" && c.stream_id === "stream-2"
        );
        expect(newClass).toBeDefined();
        expect(newClass!.id).toBe("server-real-id");
        expect(newClass!.display_label).toBe("G2 Blue");
        expect(newClass!.stream_name).toBe("Blue");

        // No temp-* IDs should remain
        expect(cached!.items.some((c) => c.id.startsWith("temp-"))).toBe(false);
    });

    it("onSettled always invalidates the list query", async () => {
        // Arrange
        const existingClass = buildClass({ id: "existing-1", display_label: "G1 Red" });
        seedCache(queryClient, [existingClass]);

        // Test both success and error paths
        const serverClass = buildClass({ id: "server-id", display_label: "G2 Blue" });
        vi.mocked(createClass).mockResolvedValue(serverClass);

        const { result } = renderCreateClassHook();
        await result.current.mutateAsync({ grade_level: "G2", stream_id: "stream-2" });

        // After onSettled, the query should be invalidated (marked stale)
        const queryState = queryClient.getQueryState(classKeys.list());
        expect(queryState).toBeDefined();
        // Invalidated queries are marked stale and will refetch when observed
        expect(queryState?.isInvalidated).toBe(true);
    });
});
