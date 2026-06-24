/**
 * Test utilities for bulk staff import tests.
 *
 * Provides:
 *   - renderWithProviders: wraps component with QueryClient + needed providers
 *   - createTestQueryClient: fresh QueryClient for each test
 *   - mockGetMe: sets up a mock for getMe() API call
 */

import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { server } from "./msw-server";

/** Create a fresh QueryClient for testing. */
export function createTestQueryClient() {
    return new QueryClient({
        defaultOptions: {
            queries: {
                retry: false,
                gcTime: 0,
                staleTime: 0,
            },
            mutations: {
                retry: false,
            },
        },
    });
}

/** Renders a component wrapped with all necessary providers. */
export function renderWithProviders(
    ui: React.ReactElement,
    options?: RenderOptions & { queryClient?: QueryClient }
) {
    const qc = options?.queryClient ?? createTestQueryClient();

    function Wrapper({ children }: { children: React.ReactNode }) {
        return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
    }

    return {
        ...render(ui, { wrapper: Wrapper, ...options }),
        queryClient: qc,
    };
}

/**
 * Register a mock for GET /api/auth/me so components can load session info.
 * Call this in beforeEach for tests that render BulkStaffImport or its children.
 */
export function mockGetMe(overrides?: Record<string, string>) {
    server.use(
        http.get("http://localhost:3000/api/auth/me", () => {
            return HttpResponse.json({
                user_id: "user-xyz",
                tenant_id: "tenant-abc",
                school_id: "school-xyz",
                full_name: "Test",
                full_name: "User",
                email: "test@school.edu",
                role: "SCHOOL_ADMIN",
                ...overrides,
            });
        })
    );
}
