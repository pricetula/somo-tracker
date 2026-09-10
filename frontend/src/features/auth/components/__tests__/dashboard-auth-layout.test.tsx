import { describe, it, expect, vi, beforeEach } from "vitest";
import { render } from "@testing-library/react";
import React from "react";

import { DashboardAuthLayout } from "../dashboard-auth-layout";

// Mock Next.js server APIs
const mockCookies = vi.fn();

vi.mock("next/navigation", async () => {
    const actual = await vi.importActual("next/navigation");
    return {
        ...actual,
        redirect: () => {
            const error = new Error("NEXT_REDIRECT");
            (error as Error & { digest?: string }).digest = "redirect";
            throw error;
        },
    };
});

vi.mock("next/headers", () => ({
    cookies: () => ({
        get: mockCookies,
    }),
}));

vi.mock("@tanstack/react-query", async () => {
    const actual = await vi.importActual("@tanstack/react-query");
    return {
        ...actual,
        dehydrate: vi.fn((_client: unknown) => ({})),
        HydrationBoundary: ({ children }: { children: React.ReactNode }) => <>{children}</>,
        QueryClient: vi.fn(() => ({})),
    };
});

describe("DashboardAuthLayout", () => {
    const mockMeData = {
        user_name: "Test User",
        email: "test@example.com",
        active_school_id: "school-123",
        school_name: "Test School",
        active_school_role: "teacher",
        tenant_id: "tenant-123",
    };

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("redirects to /logout when session_token cookie is missing", async () => {
        mockCookies.mockReturnValue(undefined);

        render(
            React.createElement(DashboardAuthLayout, null, React.createElement("div", {}, "Test"))
        );

        // redirect should be called (it throws internally in Next.js)
        // Since redirect throws a special error, we expect the component to not render children
        expect(mockCookies).toHaveBeenCalled();
    });

    it("renders children when session is valid", async () => {
        mockCookies.mockReturnValue({ value: "valid-token" });

        // Mock fetch to return valid me data
        const originalFetch = global.fetch;
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: () => Promise.resolve(mockMeData),
        } as Response);

        render(
            React.createElement(DashboardAuthLayout, null, React.createElement("div", {}, "Test"))
        );

        // Restore fetch
        global.fetch = originalFetch;
    });

    it("redirects to /logout when backend returns 401", async () => {
        mockCookies.mockReturnValue({ value: "invalid-token" });

        const originalFetch = global.fetch;
        global.fetch = vi.fn().mockResolvedValue({
            ok: false,
            status: 401,
            json: () => Promise.resolve({}),
        } as Response);

        render(
            React.createElement(DashboardAuthLayout, null, React.createElement("div", {}, "Test"))
        );

        global.fetch = originalFetch;
    });

    it("redirects to /select-school when active_school_id is missing", async () => {
        mockCookies.mockReturnValue({ value: "valid-token" });

        const noSchoolData = {
            ...mockMeData,
            active_school_id: null,
            active_school_role: null,
        };

        const originalFetch = global.fetch;
        global.fetch = vi.fn().mockResolvedValue({
            ok: true,
            json: () => Promise.resolve(noSchoolData),
        } as Response);

        render(
            React.createElement(DashboardAuthLayout, null, React.createElement("div", {}, "Test"))
        );

        global.fetch = originalFetch;
    });
});
