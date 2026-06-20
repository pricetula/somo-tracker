import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions } from "@testing-library/react";

// ─── Query Client Factory ─────────────────────────────────────────────────

/** Create a fresh QueryClient with short stale times for testing. */
export function createTestQueryClient() {
    return new QueryClient({
        defaultOptions: {
            queries: {
                retry: false, // Don't retry failed queries in tests
                gcTime: 0, // Disable garbage collection delay
                staleTime: 0, // Always refetch
            },
            mutations: {
                retry: false,
            },
        },
    });
}

// ─── Test Wrapper ─────────────────────────────────────────────────────────

interface WrapperOptions {
    queryClient?: QueryClient;
}

/**
 * Wraps a component with a QueryClientProvider for testing.
 *
 * Usage:
 *   render(<Component />, { wrapper: createWrapper() });
 *   render(<Component />, { wrapper: createWrapper({ queryClient }) });
 */
export function createWrapper(opts: WrapperOptions = {}) {
    const qc = opts.queryClient ?? createTestQueryClient();

    return function TestWrapper({ children }: { children: React.ReactNode }) {
        return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
    };
}

/**
 * Convenience: render with the test wrapper pre-applied.
 */
export function renderWithQuery(ui: React.ReactElement, options?: RenderOptions & WrapperOptions) {
    const { queryClient, ...renderOptions } = options ?? {};
    const wrapper = createWrapper({ queryClient });

    return {
        ...render(ui, { wrapper, ...renderOptions }),
        queryClient: queryClient ?? createWrapper({ queryClient }),
    };
}
