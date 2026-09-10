import { describe, it, expect, vi } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

import { useMeSession } from "../use-me-session";
import { api } from "@/lib/api/client";

// Mock the API client
vi.mock("@/lib/api/client", () => ({
    api: {
        get: vi.fn(),
    },
}));

const mockMeData = {
    user_name: "Test User",
    email: "test@example.com",
    active_school_id: "12345678-1234-1234-1234-123456789012",
    school_name: "Test School",
    active_school_role: "teacher",
    tenant_id: "tenant-123",
};

describe("useMeSession", () => {
    it("fetches and returns me data", async () => {
        (api.get as unknown as ReturnType<typeof vi.fn>).mockResolvedValueOnce(mockMeData);

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: {
                    retry: false,
                    staleTime: 0,
                },
            },
        });

        const wrapper = ({ children }: { children: React.ReactNode }) => (
            <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
        );

        const { result } = renderHook(() => useMeSession(), { wrapper });

        await waitFor(() => expect(result.current.data).toEqual(mockMeData));

        expect(api.get).toHaveBeenCalledWith("/api/me");
        expect(result.current.data).toEqual(mockMeData);
    });

    it("returns error when API fails", async () => {
        const mockError = new Error("Network error");
        (api.get as unknown as ReturnType<typeof vi.fn>).mockRejectedValueOnce(mockError);

        const queryClient = new QueryClient({
            defaultOptions: {
                queries: {
                    retry: false,
                    staleTime: 0,
                },
            },
        });

        const wrapper = ({ children }: { children: React.ReactNode }) => (
            <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
        );

        const { result } = renderHook(() => useMeSession(), { wrapper });

        await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 3000 });

        expect(result.current.error).toEqual(mockError);
    });
});
