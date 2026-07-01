/**
 * Tests for the useActiveAcademicYear and useActiveTerm hooks.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../setup/msw-server";
import { renderWithProviders } from "../setup/test-utils";
import * as React from "react";

import { useActiveAcademicYear, useActiveTerm } from "@/features/assessment";

// ─── Test helpers ─────────────────────────────────────────────────────────

function renderHook(Component: React.ComponentType) {
    return renderWithProviders(React.createElement(Component));
}

const API_BASE = "http://localhost:3000";

describe("useActiveAcademicYear", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("returns the current academic year when one is marked is_current", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/academic-years`, () => {
                return HttpResponse.json({
                    data: [
                        {
                            id: "year-2024",
                            name: "2024",
                            start_date: "2024-01-01",
                            end_date: "2024-12-31",
                            is_current: false,
                            version: 1,
                            created_at: "2024-01-01T00:00:00Z",
                        },
                        {
                            id: "year-2025",
                            name: "2025",
                            start_date: "2025-01-01",
                            end_date: "2025-12-31",
                            is_current: true,
                            version: 1,
                            created_at: "2025-01-01T00:00:00Z",
                        },
                        {
                            id: "year-2026",
                            name: "2026",
                            start_date: "2026-01-01",
                            end_date: "2026-12-31",
                            is_current: false,
                            version: 1,
                            created_at: "2026-01-01T00:00:00Z",
                        },
                    ],
                });
            })
        );

        function TestComponent() {
            const { data, isLoading, isError } = useActiveAcademicYear();
            if (isLoading) return <div>Loading...</div>;
            if (isError) return <div>Error</div>;
            if (!data) return <div>No active year</div>;
            return <div data-testid="year-id">{data.id}</div>;
        }

        renderHook(TestComponent);

        await waitFor(() => {
            expect(screen.getByTestId("year-id")).toHaveTextContent("year-2025");
        });
    });

    it("returns the first year when no year is marked current", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/academic-years`, () => {
                return HttpResponse.json({
                    data: [
                        {
                            id: "year-2024",
                            name: "2024",
                            start_date: "2024-01-01",
                            end_date: "2024-12-31",
                            is_current: false,
                            version: 1,
                            created_at: "2024-01-01T00:00:00Z",
                        },
                        {
                            id: "year-2025",
                            name: "2025",
                            start_date: "2025-01-01",
                            end_date: "2025-12-31",
                            is_current: false,
                            version: 1,
                            created_at: "2025-01-01T00:00:00Z",
                        },
                    ],
                });
            })
        );

        function TestComponent() {
            const { data, isLoading, isError } = useActiveAcademicYear();
            if (isLoading) return <div>Loading...</div>;
            if (isError) return <div>Error</div>;
            if (!data) return <div>No year</div>;
            return <div data-testid="year-id">{data.id}</div>;
        }

        renderHook(TestComponent);

        await waitFor(() => {
            expect(screen.getByTestId("year-id")).toHaveTextContent("year-2024");
        });
    });

    it("returns null when there are no academic years", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/academic-years`, () => {
                return HttpResponse.json({ data: [] });
            })
        );

        function TestComponent() {
            const { data, isLoading, isError } = useActiveAcademicYear();
            if (isLoading) return <div>Loading...</div>;
            if (isError) return <div>Error</div>;
            if (!data) return <div data-testid="null-result">null</div>;
            return <div>Has data</div>;
        }

        renderHook(TestComponent);

        await waitFor(() => {
            expect(screen.getByTestId("null-result")).toHaveTextContent("null");
        });
    });

    it("handles API errors gracefully", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/academic-years`, () => {
                return new HttpResponse(null, { status: 500 });
            })
        );

        function TestComponent() {
            const { isLoading, isError } = useActiveAcademicYear();
            if (isLoading) return <div>Loading...</div>;
            if (isError) return <div data-testid="error-state">Error</div>;
            return <div>Loaded</div>;
        }

        renderHook(TestComponent);

        await waitFor(() => {
            expect(screen.getByTestId("error-state")).toBeInTheDocument();
        });
    });
});

describe("useActiveTerm", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("returns the current term for the given academic year", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/academic-terms`, ({ request }) => {
                const url = new URL(request.url);
                expect(url.searchParams.get("academic_year_id")).toBe("year-2025");
                return HttpResponse.json({
                    data: [
                        {
                            id: "term-1",
                            academic_year_id: "year-2025",
                            name: "Term 1",
                            term_number: 1,
                            start_date: "2025-01-01",
                            end_date: "2025-04-30",
                            is_current: true,
                            is_final: false,
                            version: 1,
                            created_at: "2025-01-01T00:00:00Z",
                        },
                        {
                            id: "term-2",
                            academic_year_id: "year-2025",
                            name: "Term 2",
                            term_number: 2,
                            start_date: "2025-05-01",
                            end_date: "2025-08-31",
                            is_current: false,
                            is_final: false,
                            version: 1,
                        },
                    ],
                });
            })
        );

        function TestComponent() {
            const { data, isLoading, isError } = useActiveTerm("year-2025");
            if (isLoading) return <div>Loading...</div>;
            if (isError) return <div>Error</div>;
            if (!data) return <div>No term</div>;
            return <div data-testid="term-id">{data.id}</div>;
        }

        renderHook(TestComponent);

        await waitFor(() => {
            expect(screen.getByTestId("term-id")).toHaveTextContent("term-1");
        });
    });

    it("is disabled when yearId is undefined — does not fetch", async () => {
        function TestComponent() {
            const { data, isLoading, isFetching } = useActiveTerm(undefined);
            const dataStr = data === null ? "null" : data === undefined ? "undefined" : "has-value";
            return (
                <div>
                    <span data-testid="loading">{String(isLoading)}</span>
                    <span data-testid="fetching">{String(isFetching)}</span>
                    <span data-testid="data">{dataStr}</span>
                </div>
            );
        }

        renderHook(TestComponent);

        // When the query is disabled, loading/fetching should be false and data is undefined
        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("fetching")).toHaveTextContent("false");
        expect(screen.getByTestId("data")).toHaveTextContent("undefined");
    });

    it("returns first term when no term is marked current", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/academic-terms`, () => {
                return HttpResponse.json({
                    data: [
                        {
                            id: "term-1",
                            academic_year_id: "year-2025",
                            name: "Term 1",
                            term_number: 1,
                            start_date: "2025-01-01",
                            end_date: "2025-04-30",
                            is_current: false,
                            is_final: false,
                            version: 1,
                        },
                        {
                            id: "term-2",
                            academic_year_id: "year-2025",
                            name: "Term 2",
                            term_number: 2,
                            start_date: "2025-05-01",
                            end_date: "2025-08-31",
                            is_current: false,
                            is_final: false,
                            version: 1,
                        },
                    ],
                });
            })
        );

        function TestComponent() {
            const { data, isLoading } = useActiveTerm("year-2025");
            if (isLoading) return <div>Loading...</div>;
            if (!data) return <div>No term</div>;
            return <div data-testid="term-id">{data.id}</div>;
        }

        renderHook(TestComponent);

        await waitFor(() => {
            expect(screen.getByTestId("term-id")).toHaveTextContent("term-1");
        });
    });
});
