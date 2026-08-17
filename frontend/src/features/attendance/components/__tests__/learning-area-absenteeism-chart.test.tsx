import { render, screen } from "@testing-library/react";
import { LearningAreaAbsenteeismChart } from "../learning-area-absenteeism-chart";
import type { LearningAreaAttendanceBreakdownList } from "@/lib/api/attendance";

// ---------------------------------------------------------------------------
// Mock the data hooks so the test runs without a QueryClientProvider and the
// chart's current-term resolution is deterministic.
// ---------------------------------------------------------------------------
const mockUseCurrentTermId = vi.hoisted(() => vi.fn());
const mockUseLearningAreaAttendanceBreakdowns = vi.hoisted(() => vi.fn());

vi.mock("@/features/attendance/hooks/use-learning-area-attendance-breakdowns", () => ({
    useCurrentTermId: mockUseCurrentTermId,
    useLearningAreaAttendanceBreakdowns: mockUseLearningAreaAttendanceBreakdowns,
}));

const breakdown: LearningAreaAttendanceBreakdownList = {
    items: [
        {
            learning_area_id: "la_002",
            learning_area_name: "English",
            periods_total: 120,
            periods_present: 90,
            periods_absent: 25,
            periods_excused: 5,
            attendance_percentage: 75.0,
        },
        {
            learning_area_id: "la_001",
            learning_area_name: "Mathematics",
            periods_total: 180,
            periods_present: 160,
            periods_absent: 12,
            periods_excused: 8,
            attendance_percentage: 88.89,
        },
    ],
    total: 2,
};

describe("LearningAreaAbsenteeismChart", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // By default the active term resolves immediately and data loads.
        mockUseCurrentTermId.mockReturnValue({
            data: "term_001",
            isPending: false,
            isError: false,
            error: null,
        });
        mockUseLearningAreaAttendanceBreakdowns.mockReturnValue({
            data: breakdown,
            isLoading: false,
            isError: false,
            error: null,
        });
    });

    it("renders the heading", () => {
        render(<LearningAreaAbsenteeismChart termId="term_001" />);
        expect(
            screen.getByRole("heading", {
                name: /learning area attendance: present vs\. absent vs\. excused breakdown/i,
            })
        ).toBeInTheDocument();
    });

    it("renders a skeleton while loading", () => {
        mockUseLearningAreaAttendanceBreakdowns.mockReturnValue({
            data: undefined,
            isLoading: true,
            isError: false,
            error: null,
        });

        render(<LearningAreaAbsenteeismChart termId="term_001" />);
        expect(screen.getByRole("heading")).toBeInTheDocument();
        expect(document.querySelector(".animate-pulse")).not.toBeNull();
    });

    it("shows an alert when the fetch fails", () => {
        mockUseLearningAreaAttendanceBreakdowns.mockReturnValue({
            data: undefined,
            isLoading: false,
            isError: true,
            error: new Error("boom"),
        });

        render(<LearningAreaAbsenteeismChart termId="term_001" />);
        expect(screen.getByRole("alert")).toBeInTheDocument();
        expect(screen.getByText(/boom/)).toBeInTheDocument();
    });

    it("shows an empty-state message when there are no summaries yet", () => {
        mockUseLearningAreaAttendanceBreakdowns.mockReturnValue({
            data: { items: [], total: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });

        render(<LearningAreaAbsenteeismChart termId="term_001" />);
        expect(
            screen.getByText(/no learning area attendance summaries for this term yet/i)
        ).toBeInTheDocument();
    });

    it("renders the grouped bar chart with data", () => {
        const { container } = render(<LearningAreaAbsenteeismChart termId="term_001" />);

        // One bar rectangle per (learning area × status) — 2 areas × 3 series.
        expect(container.querySelectorAll(".recharts-bar-rectangle").length).toBe(6);
    });

    it("resolves the active term when no term id is passed", () => {
        render(<LearningAreaAbsenteeismChart />);
        expect(mockUseCurrentTermId).toHaveBeenCalledWith(true);
    });
});
