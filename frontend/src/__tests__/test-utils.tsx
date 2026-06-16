import * as React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderOptions } from "@testing-library/react";
import type { AcademicYear, CalendarState } from "@/features/calendar/types";
import type { ClassItem, ClassStreamState } from "@/features/classes/types";

// ─── Query Client Factory ─────────────────────────────────────────────────

/** Create a fresh QueryClient with short stale times for testing. */
export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,       // Don't retry failed queries in tests
        gcTime: 0,          // Disable garbage collection delay
        staleTime: 0,       // Always refetch
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
    return (
      <QueryClientProvider client={qc}>
        {children}
      </QueryClientProvider>
    );
  };
}

/**
 * Convenience: render with the test wrapper pre-applied.
 */
export function renderWithQuery(
  ui: React.ReactElement,
  options?: RenderOptions & WrapperOptions,
) {
  const { queryClient, ...renderOptions } = options ?? {};
  const wrapper = createWrapper({ queryClient });

  return {
    ...render(ui, { wrapper, ...renderOptions }),
    queryClient: queryClient ?? createWrapper({ queryClient }),
  };
}

// ─── Mock Data Factories ──────────────────────────────────────────────────

/** Create a mock AcademicYear for testing. */
export function createMockAcademicYear(overrides?: Partial<AcademicYear>): AcademicYear {
  return {
    id: "mock-year-001",
    year: 2026,
    periods: [
      {
        id: "mock-term-1",
        name: "Term 1",
        start_date: "2026-01-06",
        end_date: "2026-04-11",
        is_final: false,
      },
      {
        id: "mock-term-2",
        name: "Term 2",
        start_date: "2026-05-05",
        end_date: "2026-08-15",
        is_final: false,
      },
      {
        id: "mock-term-3",
        name: "Term 3",
        start_date: "2026-09-01",
        end_date: "2026-11-28",
        is_final: true,
      },
    ],
    ...overrides,
  };
}

/** Create mock ClassItem array for testing. */
export function createMockClasses(count = 8, stream = "East"): ClassItem[] {
  return Array.from({ length: count }, (_, i) => ({
    id: `mock-class-${i + 1}`,
    tenant_id: "mock-tenant-001",
    school_id: "mock-school-001",
    academic_year_id: "mock-year-001",
    education_system_id: "11111111-1111-1111-1111-111111111111",
    grade_id: `mock-grade-${(i % 4) + 1}`,
    name: `Grade ${(i % 4) + 1} ${stream}`,
    stream,
    is_active: true,
  }));
}

// ─── Mock Calendar Evaluator Return Values ────────────────────────────────

export const mockCalendarStates = {
  loading: { type: "loading" } as CalendarState,
  form: { type: "form", mode: "setup" } as CalendarState,
  hidden: { type: "hidden" } as CalendarState,
  prepMode: { type: "hidden" as const, alert: "prep-mode" as const },
};

// ─── Mock Class Stream Evaluator Return Values ────────────────────────────

export const mockClassStreamStates = {
  loading: { type: "loading" } as ClassStreamState,
  setup: { type: "setup" } as ClassStreamState,
  ready: { type: "ready" } as ClassStreamState,
};
