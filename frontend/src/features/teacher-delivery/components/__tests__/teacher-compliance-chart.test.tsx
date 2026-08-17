import { render, screen } from "@testing-library/react";
import { TeacherComplianceChart } from "../teacher-compliance-chart";
import type { TeacherDeliveryBreakdownList } from "@/lib/api/teacher-delivery";

// ---------------------------------------------------------------------------
// Mock the data hooks so the test runs without a QueryClientProvider and the
// chart's current-term resolution is deterministic.
// ---------------------------------------------------------------------------
const mockUseCurrentTermId = vi.hoisted(() => vi.fn());
const mockUseTeacherDeliveryBreakdown = vi.hoisted(() => vi.fn());

vi.mock("@/features/teacher-delivery/hooks/use-teacher-delivery-breakdown", () => ({
    useCurrentTermId: mockUseCurrentTermId,
    useTeacherDeliveryBreakdown: mockUseTeacherDeliveryBreakdown,
}));

const breakdown: TeacherDeliveryBreakdownList = {
    items: [
        {
            teacher_id: "teacher_002",
            teacher_name: "Teacher Two",
            tsc_number: "TSC-002",
            total_assigned_slots: 150,
            marked_slots: 145,
            missed_slots: 5,
            on_time_submission_rate: 1.0,
        },
        {
            teacher_id: "teacher_001",
            teacher_name: "Teacher One",
            tsc_number: null,
            total_assigned_slots: 180,
            marked_slots: 172,
            missed_slots: 8,
            on_time_submission_rate: 1.0,
        },
    ],
    total: 2,
};

describe("TeacherComplianceChart", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // By default the active term resolves immediately and data loads.
        mockUseCurrentTermId.mockReturnValue({
            data: "term_001",
            isPending: false,
            isError: false,
            error: null,
        });
        mockUseTeacherDeliveryBreakdown.mockReturnValue({
            data: breakdown,
            isLoading: false,
            isError: false,
            error: null,
        });
    });

    it("renders the heading", () => {
        render(<TeacherComplianceChart termId="term_001" />);
        expect(
            screen.getByRole("heading", {
                name: /teacher delivery: marked vs\. missed slots/i,
            })
        ).toBeInTheDocument();
    });

    it("renders a skeleton while loading", () => {
        mockUseTeacherDeliveryBreakdown.mockReturnValue({
            data: undefined,
            isLoading: true,
            isError: false,
            error: null,
        });

        render(<TeacherComplianceChart termId="term_001" />);
        expect(screen.getByRole("heading")).toBeInTheDocument();
        expect(document.querySelector(".animate-pulse")).not.toBeNull();
    });

    it("shows an alert when the fetch fails", () => {
        mockUseTeacherDeliveryBreakdown.mockReturnValue({
            data: undefined,
            isLoading: false,
            isError: true,
            error: new Error("boom"),
        });

        render(<TeacherComplianceChart termId="term_001" />);
        expect(screen.getByRole("alert")).toBeInTheDocument();
        expect(screen.getByText(/boom/)).toBeInTheDocument();
    });

    it("shows an empty-state message when there are no summaries yet", () => {
        mockUseTeacherDeliveryBreakdown.mockReturnValue({
            data: { items: [], total: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });

        render(<TeacherComplianceChart termId="term_001" />);
        expect(screen.getByText(/no delivery summaries for this term yet/i)).toBeInTheDocument();
    });

    it("renders the grouped bar chart with data", () => {
        const { container } = render(<TeacherComplianceChart termId="term_001" />);

        // One bar rectangle per (teacher × status) — 2 teachers × 2 series.
        expect(container.querySelectorAll(".recharts-bar-rectangle").length).toBe(4);
    });

    it("resolves the active term when no term id is passed", () => {
        render(<TeacherComplianceChart />);
        expect(mockUseCurrentTermId).toHaveBeenCalledWith(true);
    });
});
