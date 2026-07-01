/**
 * Tests for the assessment TanStack Query hooks.
 *
 * Tests the query hooks by rendering wrapper components that use them
 * and asserting against MSW-mocked API responses.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "../setup/msw-server";
import { renderWithProviders } from "../setup/test-utils";
import * as React from "react";

import {
    useBlueprints,
    useBlueprintDetail,
    useSessions,
    useSessionDetail,
    useSessionResults,
    useWeightConfigs,
} from "@/features/assessment";
import type {
    ListBlueprintsResponse,
    BlueprintDetailResponse,
    ListSessionsResponse,
    SessionDetailResponse,
    ListResultsResponse,
    ListWeightConfigsResponse,
} from "@/features/assessment/types";
import {
    buildBlueprint,
    buildBlueprintDetail,
    buildSession,
    buildResult,
} from "../factories/assessment";

// ─── Helper component to render a hook and surface its state ──────────────

// ─── MSW handlers ─────────────────────────────────────────────────────────

const API_BASE = "http://localhost:3000";

describe("useBlueprints", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("fetches and returns a list of blueprints", async () => {
        const blueprints = [buildBlueprint({ id: "bp-1", title: "Math Assessment" })];

        server.use(
            http.get(`${API_BASE}/api/v1/assessment/blueprints`, ({ request }) => {
                const url = new URL(request.url);
                expect(url.searchParams.toString()).toBe("");
                return HttpResponse.json<ListBlueprintsResponse>({ data: blueprints });
            })
        );

        function TestComponent() {
            const query = useBlueprints();
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error: {query.error?.message}</div>;
            return <div data-testid="count">{query.data?.data.length}</div>;
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("count")).toHaveTextContent("1");
        });
    });

    it("passes filter parameters to the API", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/assessment/blueprints`, ({ request }) => {
                const url = new URL(request.url);
                expect(url.searchParams.get("grade_level")).toBe("G4");
                expect(url.searchParams.get("type")).toBe("Formative_Classroom");
                return HttpResponse.json<ListBlueprintsResponse>({ data: [] });
            })
        );

        function TestComponent() {
            const query = useBlueprints({ grade_level: "G4", type: "Formative_Classroom" });
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error</div>;
            return <div data-testid="done">Done</div>;
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("done")).toBeInTheDocument();
        });
    });

    it("handles API errors gracefully", async () => {
        server.use(
            http.get(`${API_BASE}/api/v1/assessment/blueprints`, () => {
                return HttpResponse.json(
                    { code: "server_error", message: "Internal server error" },
                    { status: 500 }
                );
            })
        );

        function TestComponent() {
            const query = useBlueprints();
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div data-testid="error-msg">Error loading blueprints</div>;
            return <div>Data</div>;
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("error-msg")).toBeInTheDocument();
        });
    });
});

describe("useBlueprintDetail", () => {
    it("fetches blueprint detail with linked indicators", async () => {
        const detail = buildBlueprintDetail({ id: "bp-1" }, 2);

        server.use(
            http.get(`${API_BASE}/api/v1/assessment/blueprints/bp-1`, () => {
                return HttpResponse.json<BlueprintDetailResponse>({ data: detail });
            })
        );

        function TestComponent() {
            const query = useBlueprintDetail("bp-1");
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error</div>;
            return (
                <div>
                    <span data-testid="indicators">{query.data?.data.indicators.length}</span>
                </div>
            );
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("indicators")).toHaveTextContent("2");
        });
    });

    it("returns empty when no id is provided", () => {
        function TestComponent() {
            const query = useBlueprintDetail("", { enabled: false });
            return (
                <div>
                    <span data-testid="loading">{String(query.isLoading)}</span>
                    <span data-testid="fetching">{String(query.isFetching)}</span>
                </div>
            );
        }

        renderWithProviders(React.createElement(TestComponent));

        expect(screen.getByTestId("loading")).toHaveTextContent("false");
        expect(screen.getByTestId("fetching")).toHaveTextContent("false");
    });
});

describe("useSessions", () => {
    it("fetches and returns a list of sessions", async () => {
        const sessions = [buildSession({ id: "s-1", blueprint_id: "bp-1", class_id: "class-1" })];

        server.use(
            http.get(`${API_BASE}/api/v1/assessment/sessions`, () => {
                return HttpResponse.json<ListSessionsResponse>({ data: sessions });
            })
        );

        function TestComponent() {
            const query = useSessions();
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error</div>;
            return <div data-testid="count">{query.data?.data.length}</div>;
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("count")).toHaveTextContent("1");
        });
    });
});

describe("useSessionDetail", () => {
    it("fetches session detail with results", async () => {
        const results = [buildResult(), buildResult()];
        const session = buildSession({ id: "s-1" });

        server.use(
            http.get(`${API_BASE}/api/v1/assessment/sessions/s-1`, () => {
                return HttpResponse.json<SessionDetailResponse>({
                    data: { ...session, results },
                });
            })
        );

        function TestComponent() {
            const query = useSessionDetail("s-1");
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error</div>;
            return (
                <div>
                    <span data-testid="results">{query.data?.data.results.length}</span>
                </div>
            );
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("results")).toHaveTextContent("2");
        });
    });
});

describe("useSessionResults", () => {
    it("fetches results for a specific session", async () => {
        const results = [buildResult({ rubric_level: "EE" }), buildResult({ rubric_level: "ME" })];

        server.use(
            http.get(`${API_BASE}/api/v1/assessment/sessions/s-1/results`, () => {
                return HttpResponse.json<ListResultsResponse>({ data: results });
            })
        );

        function TestComponent() {
            const query = useSessionResults("s-1");
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error</div>;
            return <div data-testid="count">{query.data?.data.length}</div>;
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("count")).toHaveTextContent("2");
        });
    });
});

describe("useWeightConfigs", () => {
    it("fetches weight configs with optional filters", async () => {
        const configs = [
            {
                id: "wc-1",
                grade_level: "G4",
                assessment_type_code: "KNEC_Written_Assessment",
                target_exam: "KPSEA",
                weight_percent: "25.00",
                effective_from: 2026,
            },
        ];

        server.use(
            http.get(`${API_BASE}/api/v1/assessment/weight-configs`, ({ request }) => {
                const url = new URL(request.url);
                expect(url.searchParams.get("grade_level")).toBe("G4");
                return HttpResponse.json<ListWeightConfigsResponse>({ data: configs });
            })
        );

        function TestComponent() {
            const query = useWeightConfigs({ grade_level: "G4" });
            if (query.isLoading) return <div>Loading...</div>;
            if (query.isError) return <div>Error</div>;
            return <div data-testid="count">{query.data?.data.length}</div>;
        }

        renderWithProviders(React.createElement(TestComponent));

        await waitFor(() => {
            expect(screen.getByTestId("count")).toHaveTextContent("1");
        });
    });
});
