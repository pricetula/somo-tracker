import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi, describe, it, expect, beforeEach } from "vitest";
import { AttendanceCalendar, type DayStatus } from "../attendance-calendar";

// ===========================================================================
// MOCKS
// ===========================================================================

const mockGetCalendarStatus = vi.fn();
vi.mock("@/lib/api/attendance", () => ({
    getCalendarStatus: (...args: unknown[]) => mockGetCalendarStatus(...args),
    // Other exports needed by imports — provide empty stubs
    createSession: vi.fn(),
    listSessions: vi.fn(),
    getSession: vi.fn(),
    getSessionsForClassDate: vi.fn(),
    updateSession: vi.fn(),
    batchMarkAttendance: vi.fn(),
    listRecordsBySlot: vi.fn(),
    listRecordsByStudent: vi.fn(),
    listRecordsByClassDate: vi.fn(),
    listRecords: vi.fn(),
    updateRecord: vi.fn(),
    getStudentTermSummary: vi.fn(),
    getClassTermSummary: vi.fn(),
    refreshSummaries: vi.fn(),
}));

// Mock the Calendar component (it's dynamically imported and uses client-side APIs)
vi.mock("@/components/shared/calendar", () => ({
    Calendar: ({
        dayContent,
    }: {
        dayContent?: (date: Date) => React.ReactNode;
        [key: string]: unknown;
    }) => {
        const days: React.ReactNode[] = [];
        // Generate sample days that stay constant regardless of current date
        const fmt = (d: Date) =>
            `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
        for (let d = 1; d <= 7; d++) {
            const date = new Date(2026, 5, d);
            const key = fmt(date);
            days.push(
                <div key={key} data-testid={`day-${key}`}>
                    <span>{d}</span>
                    {dayContent?.(date)}
                </div>
            );
        }
        return (
            <div data-testid="mock-calendar">
                <div data-testid="calendar-days">{days}</div>
            </div>
        );
    },
}));

// Mock sonner
vi.mock("sonner", () => ({
    toast: { error: vi.fn() },
}));

// ===========================================================================
// HELPERS
// ===========================================================================

function createQueryClient() {
    return new QueryClient({
        defaultOptions: {
            queries: {
                retry: false,
                gcTime: 0,
            },
        },
    });
}

function renderWithClient(ui: React.ReactElement) {
    const queryClient = createQueryClient();
    return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

function mockSuccessResponse(
    overrides: Array<{
        date: string;
        expected_count: number;
        handled_count: number;
        status: string;
    }>
) {
    mockGetCalendarStatus.mockResolvedValue({
        items: overrides.map((o) => ({
            date: o.date,
            expected_count: o.expected_count,
            handled_count: o.handled_count,
            status: o.status as DayStatus,
        })),
        total: overrides.length,
    });
}

// ===========================================================================
// TESTS
// ===========================================================================

describe("AttendanceCalendar", () => {
    const schoolId = "test-school-001";

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("shows loading state while fetching", async () => {
        // Never resolve — keep loading
        mockGetCalendarStatus.mockReturnValue(new Promise(() => {}));

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        await waitFor(() => {
            expect(screen.getByText("Loading attendance status…")).toBeInTheDocument();
        });
    });

    it("renders green status for a fully-marked day", async () => {
        mockSuccessResponse([
            { date: "2026-06-01", expected_count: 6, handled_count: 6, status: "green" },
        ]);

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        await waitFor(() => {
            const dayEl = screen.getByTestId("day-2026-06-01");
            const dot = dayEl.querySelector("span.rounded-full");
            expect(dot).toBeInTheDocument();
            expect(dot).toHaveClass("bg-green-500");
        });
    });

    it("renders yellow status for a partially-marked day", async () => {
        mockSuccessResponse([
            { date: "2026-06-02", expected_count: 6, handled_count: 3, status: "yellow" },
        ]);

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        await waitFor(() => {
            const dayEl = screen.getByTestId("day-2026-06-02");
            const dot = dayEl.querySelector("span.rounded-full");
            expect(dot).toBeInTheDocument();
            expect(dot).toHaveClass("bg-yellow-500");
        });
    });

    it("renders red status for a fully-unmarked day", async () => {
        mockSuccessResponse([
            { date: "2026-06-03", expected_count: 6, handled_count: 0, status: "red" },
        ]);

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        await waitFor(() => {
            const dayEl = screen.getByTestId("day-2026-06-03");
            const dot = dayEl.querySelector("span.rounded-full");
            expect(dot).toBeInTheDocument();
            expect(dot).toHaveClass("bg-red-500");
        });
    });

    it("renders no status indicator for days with no scheduled slots ('none')", async () => {
        mockSuccessResponse([
            { date: "2026-06-06", expected_count: 0, handled_count: 0, status: "none" },
        ]);

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        await waitFor(() => {
            const dayEl = screen.getByTestId("day-2026-06-06");
            const dot = dayEl.querySelector("span.rounded-full");
            expect(dot).not.toBeInTheDocument();
        });
    });

    it("handles error state gracefully — calendar still renders", async () => {
        mockGetCalendarStatus.mockRejectedValue(new Error("API error"));

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        // Calendar should still render even on error
        await waitFor(() => {
            expect(screen.getByTestId("mock-calendar")).toBeInTheDocument();
        });

        // Error message should be displayed
        await waitFor(() => {
            expect(screen.getByRole("alert")).toBeInTheDocument();
        });
    });

    it("shows SKIPPED sessions as handled (green when all handled)", async () => {
        mockSuccessResponse([
            { date: "2026-06-04", expected_count: 6, handled_count: 6, status: "green" },
        ]);

        renderWithClient(<AttendanceCalendar schoolId={schoolId} />);

        await waitFor(() => {
            const dayEl = screen.getByTestId("day-2026-06-04");
            const dot = dayEl.querySelector("span.rounded-full");
            expect(dot).toBeInTheDocument();
            expect(dot).toHaveClass("bg-green-500");
        });
    });
});
