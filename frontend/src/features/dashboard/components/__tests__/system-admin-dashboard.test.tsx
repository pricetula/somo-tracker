import { render, screen } from "@testing-library/react";
import { SystemAdminDashboardPage } from "../system-admin-dashboard";
import { AcademicYearHandler } from "@/features/academic-years/components/academic-year-handler";

// ---------------------------------------------------------------------------
// Mock all data hooks so the page renders without a QueryClientProvider and
// every dashboard section resolves to a deterministic, non-erroring state.
// ---------------------------------------------------------------------------
vi.mock("@/features/academicyears/components/academic-year-handler", () => ({
    AcademicYearHandler: vi.fn(),
}));

const mockUseMemberCounts = vi.hoisted(() => vi.fn());
vi.mock("@/features/dashboard/hooks/use-member-counts", () => ({
    useMemberCounts: mockUseMemberCounts,
}));

const mockUseSchoolAttendanceKPIs = vi.hoisted(() => vi.fn());
vi.mock("@/features/attendance/hooks/use-school-attendance-kpis", () => ({
    useSchoolAttendanceKPIs: mockUseSchoolAttendanceKPIs,
}));

const mockAttendanceUseCurrentTermId = vi.hoisted(() => vi.fn());
const mockUseClassAttendanceBreakdowns = vi.hoisted(() => vi.fn());
vi.mock("@/features/attendance/hooks/use-class-attendance-breakdowns", () => ({
    useCurrentTermId: mockAttendanceUseCurrentTermId,
    useClassAttendanceBreakdowns: mockUseClassAttendanceBreakdowns,
}));

const mockLearningAreaUseCurrentTermId = vi.hoisted(() => vi.fn());
const mockUseLearningAreaAttendanceBreakdowns = vi.hoisted(() => vi.fn());
vi.mock("@/features/attendance/hooks/use-learning-area-attendance-breakdowns", () => ({
    useCurrentTermId: mockLearningAreaUseCurrentTermId,
    useLearningAreaAttendanceBreakdowns: mockUseLearningAreaAttendanceBreakdowns,
}));

const mockTeacherDeliveryUseCurrentTermId = vi.hoisted(() => vi.fn());
const mockUseTeacherDeliveryBreakdown = vi.hoisted(() => vi.fn());
vi.mock("@/features/teacher-delivery/hooks/use-teacher-delivery-breakdown", () => ({
    useCurrentTermId: mockTeacherDeliveryUseCurrentTermId,
    useTeacherDeliveryBreakdown: mockUseTeacherDeliveryBreakdown,
}));

const mockedAcademicYearHandler = vi.mocked(AcademicYearHandler);

describe("SystemAdminDashboardPage", () => {
    beforeEach(() => {
        vi.clearAllMocks();

        mockUseMemberCounts.mockReturnValue({
            data: { students: 0, teachers: 0, parents: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });
        mockUseSchoolAttendanceKPIs.mockReturnValue({
            data: {
                todays_attendance_rate: 0,
                total_present: 0,
                total_marked_records: 0,
                active_term_attendance_rate: 0,
                unmarked_slots_today: 0,
                skipped_sessions_today: 0,
            },
            isLoading: false,
            isError: false,
            error: null,
        });
        mockAttendanceUseCurrentTermId.mockReturnValue({
            data: "term_001",
            isPending: false,
            isError: false,
            error: null,
        });
        mockUseClassAttendanceBreakdowns.mockReturnValue({
            data: { items: [], total: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });
        mockLearningAreaUseCurrentTermId.mockReturnValue({
            data: "term_001",
            isPending: false,
            isError: false,
            error: null,
        });
        mockUseLearningAreaAttendanceBreakdowns.mockReturnValue({
            data: { items: [], total: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });
        mockTeacherDeliveryUseCurrentTermId.mockReturnValue({
            data: "term_001",
            isPending: false,
            isError: false,
            error: null,
        });
        mockUseTeacherDeliveryBreakdown.mockReturnValue({
            data: { items: [], total: 0 },
            isLoading: false,
            isError: false,
            error: null,
        });
    });

    test("renders the academic year handler once", () => {
        render(<SystemAdminDashboardPage />);

        expect(mockedAcademicYearHandler).toHaveBeenCalledTimes(1);
    });

    test("renders the class attendance breakdown section", () => {
        render(<SystemAdminDashboardPage />);

        expect(
            screen.getByRole("heading", {
                name: /class attendance: present vs\. late vs\. absent breakdown/i,
            })
        ).toBeInTheDocument();
    });

    test("renders the learning area attendance breakdown section", () => {
        render(<SystemAdminDashboardPage />);

        expect(
            screen.getByRole("heading", {
                name: /learning area attendance: present vs\. absent vs\. excused breakdown/i,
            })
        ).toBeInTheDocument();
    });

    test("renders the teacher compliance chart section", () => {
        render(<SystemAdminDashboardPage />);

        expect(
            screen.getByRole("heading", {
                name: /teacher delivery: marked vs\. missed slots/i,
            })
        ).toBeInTheDocument();
    });

    test("resolves the active term for all charts", () => {
        render(<SystemAdminDashboardPage />);

        expect(mockAttendanceUseCurrentTermId).toHaveBeenCalled();
        expect(mockLearningAreaUseCurrentTermId).toHaveBeenCalled();
        expect(mockTeacherDeliveryUseCurrentTermId).toHaveBeenCalled();
    });
});
