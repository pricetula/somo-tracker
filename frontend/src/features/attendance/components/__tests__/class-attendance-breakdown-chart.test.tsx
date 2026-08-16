import { render, screen } from "@testing-library/react";
import { ClassAttendanceBreakdownChart } from "../class-attendance-breakdown-chart";
import type { ClassAttendanceBreakdownList } from "@/lib/api/attendance";

// ---------------------------------------------------------------------------
// Mock the data hooks so the test runs without a QueryClientProvider and the
// chart's current-term resolution is deterministic.
// ---------------------------------------------------------------------------
const mockUseCurrentTermId = vi.hoisted(() => vi.fn());
const mockUseClassAttendanceBreakdowns = vi.hoisted(() => vi.fn());

vi.mock("@/features/attendance/hooks/use-class-attendance-breakdowns", () => ({
    useCurrentTermId: mockUseCurrentTermId,
    useClassAttendanceBreakdowns: mockUseClassAttendanceBreakdowns,
}));

const breakdown: ClassAttendanceBreakdownList = {
    items: [
        {
            class_id: "class_002",
            class_name: "G1 Green",
            total_enrolled_avg: 30,
            present_count: 20,
            late_count: 3,
            absent_count: 5,
            excused_count: 2,
            term_attendance_rate: 66.67,
        },
        {
            class_id: "class_001",
            class_name: "G1 Blue",
            total_enrolled_avg: 30,
            present_count: 25,
            late_count: 2,
            absent_count: 3,
            excused_count: 0,
            term_attendance_rate: 83.33,
        },
    ],
    total: 2,
};

describe("ClassAttendanceBreakdownChart", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // By default the active term resolves immediately and data loads.
        mockUseCurrentTermId.mockReturnValue({
            data: "term_001",
            isPending: false,
            isError: false,
            error: null,
        });
        mockUseClassAttendanceBreakdowns.mockReturnValue({
            data: breakdown,
            isLoading: false,
            isError: false,
            error: null,
        });
    });

    it("renders the heading", () => {
        render(<ClassAttendanceBreakdownChart termId="term_001" />);
        expect(
            screen.getByRole("heading", {
                name: /class attendance: present vs\. late vs\. absent breakdown/i,
            })
        ).toBeInTheDocument();
    });

    it("renders a skeleton while loading", () => {
        mockUseClassAttendanceBreakdowns.mockReturnValue({
            data: undefined,
            isLoading: true,
            isError: false,
            error: null,
        });

        render(<ClassAttendanceBreakdownChart termId="term_001" />);
        expect(screen.getByRole("heading")).toBeInTheDocument();
        expect(document.querySelector(".animate-pulse")).not.toBeNull();
    });

    it("shows an alert when the fetch fails", () => {
        mockUseClassAttendanceBreakdowns.mockReturnValue({
            data: undefined,
            isLoading: false,
            isError: true,
            error: new Error("boom"),
        });

        render(<ClassAttendanceBreakdownChart termId="term_001" />);
        expect(screen.getByRole("alert")).toBeInTheDocument();
        expect(screen.getByText(/boom/)).toBeInTheDocument();
    });

    it("shows an empty-state message when there are no summaries yet", () => {
        mockUseClassAttendanceBreakdowns.mockReturnValue({
            data: { items: [], total: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });

        render(<ClassAttendanceBreakdownChart termId="term_001" />);
        expect(screen.getByText(/no attendance summaries for this term yet/i)).toBeInTheDocument();
    });

    it("renders the grouped bar chart with data", () => {
        const { container } = render(<ClassAttendanceBreakdownChart termId="term_001" />);

        // One bar rectangle per (class × status) — 2 classes × 3 series.
        expect(container.querySelectorAll(".recharts-bar-rectangle").length).toBe(6);
    });

    it("resolves the active term when no term id is passed", () => {
        render(<ClassAttendanceBreakdownChart />);
        expect(mockUseCurrentTermId).toHaveBeenCalledWith(true);
    });
});
