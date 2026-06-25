/**
 * Tests for useParentLookup, useClassLookup, useExistingStudents, and useLookups hooks.
 *
 * Covers: loading states, successful data fetch, error handling, retry,
 * combined hook, and Map key normalization.
 *
 * Uses MSW for HTTP mocking (server imported from __tests__/setup/msw-server).
 *
 * To run: pnpm vitest run src/features/student-import/__tests__/use-lookups.test.ts
 */

import { describe, it, expect, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../../../../__tests__/setup/msw-server";
import {
    useParentLookup,
    useClassLookup,
    useExistingStudents,
    useLookups,
} from "../hooks/use-lookups";

// ─── Helpers ──────────────────────────────────────────────────────────────

const API_BASE = "http://localhost:3000";

function mockParentsEndpoint(response: unknown, status = 200) {
    server.use(
        http.get(`${API_BASE}/api/v1/parents`, () => {
            if (status !== 200) {
                return HttpResponse.json(response, { status });
            }
            return HttpResponse.json(response);
        })
    );
}

function mockClassesEndpoint(response: unknown, status = 200) {
    server.use(
        http.get(`${API_BASE}/api/v1/classes`, () => {
            if (status !== 200) {
                return HttpResponse.json(response, { status });
            }
            return HttpResponse.json(response);
        })
    );
}

function mockExistingStudentsEndpoint(response: unknown, status = 200) {
    server.use(
        http.get(`${API_BASE}/api/v1/students`, () => {
            if (status !== 200) {
                return HttpResponse.json(response, { status });
            }
            return HttpResponse.json(response);
        })
    );
}

// ─── Tests: useParentLookup ───────────────────────────────────────────────

describe("useParentLookup", () => {
    afterEach(() => {
        server.resetHandlers();
    });

    it("starts in loading state", () => {
        // No endpoint mock → request hangs → loading stays true
        const { result } = renderHook(() => useParentLookup());
        expect(result.current.parentsLoading).toBe(true);
    });

    it("transitions to loaded state with correctly keyed Map", async () => {
        mockParentsEndpoint([
            { id: "p1", full_name: "Nancy Onyinde", phone: "+254700111222" },
            { id: "p2", full_name: "John Kamau", email: "john@school.edu" },
        ]);

        const { result } = renderHook(() => useParentLookup());

        await waitFor(() => {
            expect(result.current.parentsLoading).toBe(false);
        });

        expect(result.current.parentsError).toBeNull();
        const map = result.current.parentsMap;

        // Check map keys are normalized (lowercase, no spaces)
        expect(map.get("nancyonyinde")).toBeDefined();
        expect(map.get("nancyonyinde")!.id).toBe("p1");
        expect(map.get("nancyonyinde")!.full_name).toBe("Nancy Onyinde");

        expect(map.get("johnkamau")).toBeDefined();
        expect(map.get("johnkamau")!.id).toBe("p2");
        expect(map.get("johnkamau")!.full_name).toBe("John Kamau");
    });

    it("sets error state on API failure", async () => {
        mockParentsEndpoint({ code: "server_error", message: "Internal server error" }, 500);

        const { result } = renderHook(() => useParentLookup());

        await waitFor(() => {
            expect(result.current.parentsLoading).toBe(false);
        });

        expect(result.current.parentsError).not.toBeNull();
        expect(result.current.parentsMap.size).toBe(0);
    });

    it("retry re-fetches data after error", async () => {
        // First request fails
        mockParentsEndpoint({ code: "server_error", message: "Down" }, 500);

        const { result } = renderHook(() => useParentLookup());

        await waitFor(() => {
            expect(result.current.parentsLoading).toBe(false);
        });
        expect(result.current.parentsError).not.toBeNull();

        // Second request succeeds
        mockParentsEndpoint([{ id: "p1", full_name: "Nancy Onyinde" }]);

        result.current.retryParents();

        await waitFor(() => {
            expect(result.current.parentsLoading).toBe(false);
            expect(result.current.parentsError).toBeNull();
        });

        expect(result.current.parentsMap.size).toBe(1);
        expect(result.current.parentsMap.get("nancyonyinde")?.id).toBe("p1");
    });

    it("handles network timeout gracefully", async () => {
        // Simulate a hanging request by not mocking → MSW bypasses it (unhandled)
        // which in test env means the request never resolves.
        // We just verify loading stays true initially.
        const { result } = renderHook(() => useParentLookup());
        expect(result.current.parentsLoading).toBe(true);
    });
});

// ─── Tests: useClassLookup ────────────────────────────────────────────────

describe("useClassLookup", () => {
    afterEach(() => {
        server.resetHandlers();
    });

    it("starts in loading state", () => {
        const { result } = renderHook(() => useClassLookup());
        expect(result.current.classesLoading).toBe(true);
    });

    it("transitions to loaded with normalized class names", async () => {
        mockClassesEndpoint([
            { id: "c1", name: "Class 4 West" },
            { id: "c2", name: "Grade 3 East" },
            { id: "c3", name: "Form 1 North" },
        ]);

        const { result } = renderHook(() => useClassLookup());

        await waitFor(() => {
            expect(result.current.classesLoading).toBe(false);
        });

        expect(result.current.classesError).toBeNull();
        const map = result.current.classesMap;

        // Check normalized keys
        expect(map.get("4west")).toBeDefined();
        expect(map.get("4west")!.id).toBe("c1");

        expect(map.get("3east")).toBeDefined();
        expect(map.get("3east")!.id).toBe("c2");

        expect(map.get("1north")).toBeDefined();
        expect(map.get("1north")!.id).toBe("c3");
    });

    it("sets error state on API failure", async () => {
        mockClassesEndpoint({ code: "server_error", message: "DB connection failed" }, 503);

        const { result } = renderHook(() => useClassLookup());

        await waitFor(() => {
            expect(result.current.classesLoading).toBe(false);
        });

        expect(result.current.classesError).not.toBeNull();
        expect(result.current.classesMap.size).toBe(0);
    });

    it("retry re-fetches classes after error", async () => {
        mockClassesEndpoint({ code: "error" }, 500);

        const { result } = renderHook(() => useClassLookup());

        await waitFor(() => {
            expect(result.current.classesLoading).toBe(false);
        });
        expect(result.current.classesError).not.toBeNull();

        mockClassesEndpoint([{ id: "c1", name: "Class 1 North" }]);

        result.current.retryClasses();

        await waitFor(() => {
            expect(result.current.classesError).toBeNull();
        });

        expect(result.current.classesMap.size).toBe(1);
    });
});

// ─── Tests: useExistingStudents ───────────────────────────────────────────

describe("useExistingStudents", () => {
    afterEach(() => {
        server.resetHandlers();
    });

    it("starts in loading state", () => {
        const { result } = renderHook(() => useExistingStudents());
        expect(result.current.existingStudentsLoading).toBe(true);
    });

    it("fetches existing students successfully", async () => {
        mockExistingStudentsEndpoint([
            { full_name: "Alice Wanjiku", date_of_birth: "2010-03-15", upi_number: "KP1234567A" },
            { full_name: "Bob Kimani", date_of_birth: "2011-07-22", upi_number: null },
        ]);

        const { result } = renderHook(() => useExistingStudents());

        await waitFor(() => {
            expect(result.current.existingStudentsLoading).toBe(false);
        });

        expect(result.current.existingStudentsError).toBeNull();
        expect(result.current.existingStudents).toHaveLength(2);
        expect(result.current.existingStudents[0].full_name).toBe("Alice Wanjiku");
    });

    it("returns empty array on error", async () => {
        mockExistingStudentsEndpoint({ code: "error" }, 500);

        const { result } = renderHook(() => useExistingStudents());

        await waitFor(() => {
            expect(result.current.existingStudentsLoading).toBe(false);
        });

        expect(result.current.existingStudentsError).not.toBeNull();
        expect(result.current.existingStudents).toEqual([]);
    });
});

// ─── Tests: useLookups (combined) ─────────────────────────────────────────

describe("useLookups (combined)", () => {
    afterEach(() => {
        server.resetHandlers();
    });

    it("loads all three data sources", async () => {
        mockParentsEndpoint([{ id: "p1", full_name: "Nancy Onyinde" }]);
        mockClassesEndpoint([{ id: "c1", name: "Class 4 West" }]);
        mockExistingStudentsEndpoint([{ full_name: "Alice Wanjiku", date_of_birth: "2010-03-15" }]);

        const { result } = renderHook(() => useLookups());

        await waitFor(() => {
            expect(result.current.parentsLoading).toBe(false);
            expect(result.current.classesLoading).toBe(false);
        });

        expect(result.current.parentsMap.size).toBe(1);
        expect(result.current.classesMap.size).toBe(1);
        expect(result.current.existingStudents).toHaveLength(1);
    });

    it("recovers with retry parents after failure", async () => {
        // Fail parents initially
        mockParentsEndpoint({ code: "error" }, 500);
        mockClassesEndpoint([{ id: "c1", name: "Class 1" }]);
        mockExistingStudentsEndpoint([]);

        const { result } = renderHook(() => useLookups());

        await waitFor(() => {
            expect(result.current.parentsError).not.toBeNull();
            expect(result.current.classesLoading).toBe(false);
        });

        // Retry parents with success
        mockParentsEndpoint([{ id: "p1", full_name: "Nancy" }]);
        result.current.retryParents();

        await waitFor(() => {
            expect(result.current.parentsError).toBeNull();
        });

        expect(result.current.parentsMap.size).toBe(1);
    });
});
