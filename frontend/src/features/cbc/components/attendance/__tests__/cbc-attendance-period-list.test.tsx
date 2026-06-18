/**
 * Tests for CbcAttendancePeriodList component.
 *
 * Tests filter controls, date range presets, gap display, period selection,
 * empty states, and loading state.
 */

import * as React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { CbcAttendancePeriodList } from "@/features/cbc/components/attendance/cbc-attendance-period-list";
import type { AttendancePeriodSummary, AttendanceGap } from "@/features/cbc/types";

// ─── Mock hooks (query layer) ─────────────────────────────────────────────

const mockUseSummaries = vi.fn();
const mockUseGaps = vi.fn();

vi.mock("@/features/cbc/hooks/use-cbc-attendance", () => ({
    useCbcAttendancePeriodSummaries: (...args: unknown[]) => mockUseSummaries(...args),
    useCbcAttendanceGaps: (...args: unknown[]) => mockUseGaps(...args),
}));

vi.mock("lucide-react", () => ({
    Filter: () => <span data-testid="mock-filter" />,
    RotateCcw: () => <span data-testid="mock-rotate" />,
    XIcon: () => <span data-testid="mock-x-icon" />,
    ArrowLeft: () => <span data-testid="mock-arrow-left" />,
}));

vi.mock("@/components/ui/select", () => ({
    Select: ({
        value,
        onValueChange,
        children,
    }: {
        value: string;
        onValueChange: (v: string) => void;
        children: React.ReactNode;
    }) => (
        <div data-testid="mock-select" data-value={value}>
            {children}
            <button data-testid="mock-select-trigger" onClick={() => onValueChange("all")}>
                {value === "all" ? "All learning areas" : value}
            </button>
        </div>
    ),
    SelectTrigger: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
    SelectValue: ({ placeholder }: { placeholder: string }) => <span>{placeholder}</span>,
    SelectContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
    SelectItem: ({ value, children }: { value: string; children: React.ReactNode }) => (
        <div data-value={value}>{children}</div>
    ),
}));

vi.mock("@/components/ui/badge", () => ({
    Badge: ({ children }: { children: React.ReactNode }) => (
        <span data-testid="badge">{children}</span>
    ),
}));

// ─── Default mock data ────────────────────────────────────────────────────

function createMockSummary(overrides?: Partial<AttendancePeriodSummary>): AttendancePeriodSummary {
    return {
        id: `period-${Math.random().toString(36).slice(2, 8)}`,
        date_recorded: "2026-06-18",
        cbc_learning_area_id: "area-1",
        learning_area_name: "Mathematics",
        recorded_by_name: "John Otieno",
        recorded_by_id: "teacher-1",
        recorded_at: "2026-06-18T08:00:00Z",
        total_students: 10,
        present_count: 8,
        absent_count: 1,
        late_count: 1,
        excused_count: 0,
        unmarked_count: 0,
        ...overrides,
    };
}

function createMockGap(overrides?: Partial<AttendanceGap>): AttendanceGap {
    return {
        slot_id: `slot-${Math.random().toString(36).slice(2, 8)}`,
        class_id: "class-1",
        cbc_learning_area_id: "area-1",
        learning_area_name: "Mathematics",
        day_of_week: 1,
        start_time: "08:00",
        end_time: "08:40",
        date: "2026-06-17",
        ...overrides,
    };
}

// ─── Props ────────────────────────────────────────────────────────────────

const defaultProps = {
    classId: "class-1",
    learningAreaOptions: [
        { id: "area-1", name: "Mathematics" },
        { id: "area-2", name: "English" },
    ],
    selectedPeriodId: null as string | null,
    onSelectPeriod: vi.fn(),
    onFillGap: vi.fn(),
    onStartNewPeriod: vi.fn(),
    academicTermId: "term-1",
};

// ═══════════════════════════════════════════════════════════════════════════
// TESTS
// ═══════════════════════════════════════════════════════════════════════════

describe("CbcAttendancePeriodList", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Default mock returns: no summaries, no gaps, not loading
        mockUseSummaries.mockReturnValue({ data: [], isLoading: false });
        mockUseGaps.mockReturnValue({ data: [], isLoading: false });
    });

    afterEach(() => {
        cleanup();
    });

    // ── Loading state ────────────────────────────────────────────────

    it("renders skeleton while summaries are loading", () => {
        mockUseSummaries.mockReturnValue({ data: [], isLoading: true });
        mockUseGaps.mockReturnValue({ data: [], isLoading: true });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        const skeletons = document.querySelectorAll(".animate-pulse");
        // There should be skeleton divs from the loading state
        expect(skeletons.length).toBeGreaterThanOrEqual(1);
    });

    it("does not show period rows while loading", () => {
        mockUseSummaries.mockReturnValue({ data: [], isLoading: true });
        mockUseGaps.mockReturnValue({ data: [], isLoading: true });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.queryByText("Mathematics")).toBeNull();
    });

    // ── Empty state ──────────────────────────────────────────────────

    it("shows empty message when no periods found", () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(
            screen.getByText("No attendance periods found for the selected filters.")
        ).toBeTruthy();
    });

    it("shows empty message in gaps-only view when no gaps", async () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        // Toggle gaps-only
        const gapsBtn = screen.getByText("Gaps only");
        await userEvent.click(gapsBtn);
        expect(
            screen.getByText("No gaps found — all scheduled slots have attendance recorded.")
        ).toBeTruthy();
    });

    // ── Period display ───────────────────────────────────────────────

    it("renders period summaries with learning area names", () => {
        mockUseSummaries.mockReturnValue({
            data: [
                createMockSummary({
                    id: "p1",
                    learning_area_name: "Mathematics",
                    date_recorded: "2026-06-18",
                }),
                createMockSummary({
                    id: "p2",
                    learning_area_name: "English",
                    date_recorded: "2026-06-17",
                }),
            ],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        // Use getAllByText because SelectItem children also render learning area names
        const mathElements = screen.getAllByText("Mathematics");
        expect(mathElements.length).toBeGreaterThanOrEqual(1);
        const engElements = screen.getAllByText("English");
        expect(engElements.length).toBeGreaterThanOrEqual(1);
    });

    it("shows the recorder name for each period", () => {
        mockUseSummaries.mockReturnValue({
            data: [
                createMockSummary({
                    id: "p1",
                    recorded_by_name: "John Otieno",
                }),
            ],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("by John Otieno")).toBeTruthy();
    });

    it("calls onSelectPeriod when a period row is clicked", async () => {
        const onSelectPeriod = vi.fn();
        mockUseSummaries.mockReturnValue({
            data: [createMockSummary({ id: "p1" })],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} onSelectPeriod={onSelectPeriod} />);
        // Find the button that contains "Mathematics" text in the period row (not the select option)
        // The period row has text like "by John Otieno" — use that to find the correct button
        const periodBtn = screen.getByText("by John Otieno").closest("button")!;
        await userEvent.click(periodBtn);
        expect(onSelectPeriod).toHaveBeenCalledWith("p1");
    });

    it("shows completion badge for fully marked periods", () => {
        mockUseSummaries.mockReturnValue({
            data: [
                createMockSummary({
                    id: "p1",
                    unmarked_count: 0,
                }),
            ],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("Complete")).toBeTruthy();
    });

    it("shows unmarked count for partially marked periods", () => {
        mockUseSummaries.mockReturnValue({
            data: [
                createMockSummary({
                    id: "p1",
                    unmarked_count: 3,
                }),
            ],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("3 unmarked")).toBeTruthy();
    });

    // ── Gaps display ─────────────────────────────────────────────────

    it("shows gap notification bar when there are gaps", () => {
        mockUseGaps.mockReturnValue({
            data: [createMockGap()],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("1 unattended slot in this period")).toBeTruthy();
    });

    it("shows plural gap notification for multiple gaps", () => {
        mockUseGaps.mockReturnValue({
            data: [createMockGap(), createMockGap()],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("2 unattended slots in this period")).toBeTruthy();
    });

    it("renders gap rows in gaps-only view", async () => {
        mockUseGaps.mockReturnValue({
            data: [
                createMockGap({
                    learning_area_name: "English",
                    start_time: "09:00",
                    end_time: "09:40",
                }),
            ],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        // Toggle gaps-only
        const gapsBtn = screen.getByText("Gaps only");
        await userEvent.click(gapsBtn);
        // Use getAllByText because SelectItem also renders learning area names
        const engElements = screen.getAllByText("English");
        expect(engElements.length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText("09:00–09:40 · No attendance taken")).toBeTruthy();
    });

    it("calls onFillGap when 'Take now' is clicked in gaps-only view", async () => {
        const onFillGap = vi.fn();
        const gap = createMockGap({ slot_id: "slot-1" });
        mockUseGaps.mockReturnValue({
            data: [gap],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} onFillGap={onFillGap} />);
        // Toggle gaps-only
        const gapsBtn = screen.getByText("Gaps only");
        await userEvent.click(gapsBtn);
        const takeNowBtn = screen.getByText("Take now");
        await userEvent.click(takeNowBtn);
        expect(onFillGap).toHaveBeenCalledWith(gap);
    });

    // ── Filter controls ──────────────────────────────────────────────

    it("renders date range preset buttons", () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("This week")).toBeTruthy();
        expect(screen.getByText("This month")).toBeTruthy();
        expect(screen.getByText("Last 30d")).toBeTruthy();
        expect(screen.getByText("Term")).toBeTruthy();
    });

    it("renders learning area filter", () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        // The mock Select renders both the trigger and SelectItem children
        // Use getAllByText since "All learning areas" might appear in multiple places
        const allAreasElements = screen.getAllByText("All learning areas");
        expect(allAreasElements.length).toBeGreaterThanOrEqual(1);
    });

    it("renders gaps-only toggle button", () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("Gaps only")).toBeTruthy();
    });

    it("shows gap count badge on gaps toggle", () => {
        mockUseGaps.mockReturnValue({
            data: [createMockGap(), createMockGap(), createMockGap()],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        const badge = screen.getAllByTestId("badge");
        expect(badge.length).toBeGreaterThanOrEqual(1);
    });

    // ── Period count footer ──────────────────────────────────────────

    it("shows period count footer", () => {
        mockUseSummaries.mockReturnValue({
            data: [
                createMockSummary({ id: "p1" }),
                createMockSummary({ id: "p2" }),
                createMockSummary({ id: "p3" }),
            ],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText(/3 periods/)).toBeTruthy();
    });

    it("shows singular period in footer", () => {
        mockUseSummaries.mockReturnValue({
            data: [createMockSummary({ id: "p1" })],
            isLoading: false,
        });
        render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText(/1 period/)).toBeTruthy();
    });

    // ── Filter interaction: learning area filter ─────────────────────

    it("defaults learning area filter to 'all'", () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        const select = screen.getByTestId("mock-select");
        expect(select.getAttribute("data-value")).toBe("all");
    });

    // ── Reset button ─────────────────────────────────────────────────

    it("shows reset button when gaps-only is active", async () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        const gapsBtn = screen.getByText("Gaps only");
        await userEvent.click(gapsBtn);
        expect(screen.getByText("Reset")).toBeTruthy();
    });

    it("clears filters when reset is clicked", async () => {
        render(<CbcAttendancePeriodList {...defaultProps} />);
        const gapsBtn = screen.getByText("Gaps only");
        await userEvent.click(gapsBtn);
        expect(screen.getByText("Reset")).toBeTruthy();
        await userEvent.click(screen.getByText("Reset"));
        // After reset, gaps-only should be off — no "Reset" button
        expect(screen.queryByText("Reset")).toBeNull();
    });

    // ── Response time / stale data ───────────────────────────────────

    it("updates when new summary data arrives", () => {
        const { rerender } = render(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.queryByText("Science")).toBeNull();

        mockUseSummaries.mockReturnValue({
            data: [
                createMockSummary({
                    id: "p1",
                    learning_area_name: "Science",
                }),
            ],
            isLoading: false,
        });
        rerender(<CbcAttendancePeriodList {...defaultProps} />);
        expect(screen.getByText("Science")).toBeTruthy();
    });
});
