/**
 * Tests for CbcAttendanceGrid component.
 *
 * Tests loading, empty, active, read-only states, confirm-on-edit dialog,
 * and bulk "mark remaining as Present" interactions.
 */

import * as React from "react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { CbcAttendanceGrid } from "@/features/cbc/components/attendance/cbc-attendance-grid";
import type {
    AttendanceStudentRow,
    AttendanceStatus,
    CbcAttendanceLogDetail,
    OfflineAttendanceEntry,
} from "@/features/cbc/types";

// ─── Mock child components ────────────────────────────────────────────────

vi.mock("@/features/cbc/components/attendance/cbc-attendance-student-row", () => ({
    CbcAttendanceStudentRow: ({
        studentName,
        currentStatus,
        isSaving,
        syncPending,
        onSelectStatus,
        readOnly,
    }: {
        studentName: string;
        currentStatus: AttendanceStatus | null;
        isSaving: boolean;
        syncPending: boolean;
        onSelectStatus: (status: AttendanceStatus) => void;
        readOnly: boolean;
    }) => (
        <div
            data-testid="student-row"
            data-name={studentName}
            data-status={currentStatus ?? "null"}
        >
            <span>{studentName}</span>
            {syncPending && <span data-testid="sync-pending">saving...</span>}
            {isSaving && <span data-testid="is-saving">saving in progress</span>}
            {readOnly && <span data-testid="read-only">read-only</span>}
            <button
                data-testid="btn-present"
                onClick={() => onSelectStatus("PRESENT")}
                disabled={readOnly}
            >
                Present
            </button>
            <button
                data-testid="btn-absent"
                onClick={() => onSelectStatus("ABSENT")}
                disabled={readOnly}
            >
                Absent
            </button>
        </div>
    ),
}));

vi.mock("lucide-react", () => ({
    Users: () => <span data-testid="mock-users" />,
    AlertTriangle: () => <span data-testid="mock-alert-triangle" />,
    XIcon: () => <span data-testid="mock-x-icon" />,
}));

// ─── Helpers ──────────────────────────────────────────────────────────────

function createMockStudent(overrides?: Partial<AttendanceStudentRow>): AttendanceStudentRow {
    return {
        student_id: "stu-1",
        student_name: "Alice Kimani",
        first_name: "Alice",
        last_name: "Kimani",
        status: null,
        log_id: null,
        ...overrides,
    };
}

// ─── Default props ────────────────────────────────────────────────────────

const defaultProps = {
    students: [
        createMockStudent({ student_id: "stu-1", student_name: "Alice Kimani" }),
        createMockStudent({ student_id: "stu-2", student_name: "Bob Ochieng" }),
        createMockStudent({ student_id: "stu-3", student_name: "Carol Wanjiku" }),
    ],
    logs: [] as CbcAttendanceLogDetail[],
    isLoading: false,
    periodId: "period-1",
    localQueue: [] as OfflineAttendanceEntry[],
    onSelectStatus: vi.fn(),
    onRemarksChange: vi.fn(),
    onMarkRemainingAsPresent: vi.fn(),
    savingStudentIds: new Set<string>(),
    canEdit: true,
};

// ═══════════════════════════════════════════════════════════════════════════
// TESTS
// ═══════════════════════════════════════════════════════════════════════════

describe("CbcAttendanceGrid", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    afterEach(() => {
        cleanup();
    });

    // ── Loading state ────────────────────────────────────────────────

    it("renders skeleton placeholders while loading", () => {
        render(<CbcAttendanceGrid {...defaultProps} isLoading={true} />);
        const skeletons = document.querySelectorAll(".animate-pulse");
        expect(skeletons.length).toBeGreaterThanOrEqual(1);
    });

    it("does not show student rows while loading", () => {
        render(<CbcAttendanceGrid {...defaultProps} isLoading={true} />);
        expect(screen.queryByText("Alice Kimani")).toBeNull();
    });

    // ── Empty state ──────────────────────────────────────────────────

    it("shows empty message when no students enrolled", () => {
        render(<CbcAttendanceGrid {...defaultProps} students={[]} />);
        expect(
            screen.getByText("No students enrolled in this class for the current term.")
        ).toBeTruthy();
    });

    // ── No period selected ───────────────────────────────────────────

    it("shows student names in dimmed state when no periodId", () => {
        render(<CbcAttendanceGrid {...defaultProps} periodId={null} />);
        expect(screen.getByText("Alice Kimani")).toBeTruthy();
        expect(screen.getByText("Bob Ochieng")).toBeTruthy();
        expect(screen.getByText("Carol Wanjiku")).toBeTruthy();
        // Should show hint text
        expect(screen.getByText("Select a learning area to enable marking")).toBeTruthy();
    });

    it("shows student count when no periodId", () => {
        render(<CbcAttendanceGrid {...defaultProps} periodId={null} />);
        expect(screen.getByText("3 students")).toBeTruthy();
    });

    // ── Active grid ──────────────────────────────────────────────────

    it("renders all student rows", () => {
        render(<CbcAttendanceGrid {...defaultProps} />);
        const rows = screen.getAllByTestId("student-row");
        expect(rows).toHaveLength(3);
    });

    it("shows student count header", () => {
        render(<CbcAttendanceGrid {...defaultProps} />);
        expect(screen.getByText("3 students")).toBeTruthy();
    });

    it("shows unmarked count when some students are not marked", () => {
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
            createMockStudent({ student_id: "stu-2", status: null }),
            createMockStudent({ student_id: "stu-3", status: "ABSENT", log_id: "log-2" }),
        ];
        render(<CbcAttendanceGrid {...defaultProps} students={students} />);
        expect(screen.getByText("· 1 not marked")).toBeTruthy();
    });

    it("shows 'Mark remaining as Present' button when there are unmarked students", () => {
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
            createMockStudent({ student_id: "stu-2", status: null }),
            createMockStudent({ student_id: "stu-3", status: null }),
        ];
        render(<CbcAttendanceGrid {...defaultProps} students={students} />);
        expect(screen.getByText("Mark remaining as Present")).toBeTruthy();
    });

    it("hides 'Mark remaining' button when all students are marked", () => {
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
            createMockStudent({ student_id: "stu-2", status: "ABSENT", log_id: "log-2" }),
            createMockStudent({ student_id: "stu-3", status: "LATE", log_id: "log-3" }),
        ];
        render(<CbcAttendanceGrid {...defaultProps} students={students} />);
        expect(screen.queryByText("Mark remaining as Present")).toBeNull();
    });

    it("hides 'not marked' count when all students are marked", () => {
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
            createMockStudent({ student_id: "stu-2", status: "ABSENT", log_id: "log-2" }),
            createMockStudent({ student_id: "stu-3", status: "LATE", log_id: "log-3" }),
        ];
        render(<CbcAttendanceGrid {...defaultProps} students={students} />);
        expect(screen.queryByText(/not marked/)).toBeNull();
    });

    // ── Status selection ─────────────────────────────────────────────

    it("calls onSelectStatus with periodId when a status button is clicked", async () => {
        const onSelectStatus = vi.fn();
        render(<CbcAttendanceGrid {...defaultProps} onSelectStatus={onSelectStatus} />);
        const presentBtns = screen.getAllByTestId("btn-present");
        await userEvent.click(presentBtns[0]);
        expect(onSelectStatus).toHaveBeenCalledWith("stu-1", "PRESENT", "period-1");
    });

    // ── Confirm-on-edit dialog ────────────────────────────────────────

    it("shows confirm dialog when editing an existing mark", async () => {
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
        ];
        render(<CbcAttendanceGrid {...defaultProps} students={students} />);
        const presentBtns = screen.getAllByTestId("btn-present");
        await userEvent.click(presentBtns[0]);

        // Dialog should appear
        expect(screen.getByText("Editing existing record")).toBeTruthy();
        expect(
            screen.getByText(
                "You're about to change an already-submitted attendance mark. This will update the attendance record for this student."
            )
        ).toBeTruthy();
    });

    it("cancels confirm dialog and does not call onSelectStatus", async () => {
        const onSelectStatus = vi.fn();
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
        ];
        render(
            <CbcAttendanceGrid
                {...defaultProps}
                students={students}
                onSelectStatus={onSelectStatus}
            />
        );
        const presentBtns = screen.getAllByTestId("btn-present");
        await userEvent.click(presentBtns[0]);

        // Click cancel
        const cancelBtn = screen.getByText("Cancel");
        await userEvent.click(cancelBtn);
        expect(onSelectStatus).not.toHaveBeenCalled();
        expect(screen.queryByText("Editing existing record")).toBeNull();
    });

    it("continues editing after confirming dialog", async () => {
        const onSelectStatus = vi.fn();
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
        ];
        render(
            <CbcAttendanceGrid
                {...defaultProps}
                students={students}
                onSelectStatus={onSelectStatus}
            />
        );
        const presentBtns = screen.getAllByTestId("btn-present");
        await userEvent.click(presentBtns[0]);

        // Click continue editing
        const continueBtn = screen.getByText("Continue editing");
        await userEvent.click(continueBtn);
        expect(onSelectStatus).toHaveBeenCalledWith("stu-1", "PRESENT", "period-1");
    });

    it("only shows confirm dialog once per student per session", async () => {
        const onSelectStatus = vi.fn();
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
        ];
        render(
            <CbcAttendanceGrid
                {...defaultProps}
                students={students}
                onSelectStatus={onSelectStatus}
            />
        );
        // First click — confirm dialog
        const presentBtns = screen.getAllByTestId("btn-present");
        await userEvent.click(presentBtns[0]);
        await userEvent.click(screen.getByText("Continue editing"));

        // Second click on same student — no dialog
        vi.clearAllMocks();
        await userEvent.click(presentBtns[0]);
        expect(screen.queryByText("Editing existing record")).toBeNull();
        expect(onSelectStatus).toHaveBeenCalledWith("stu-1", "PRESENT", "period-1");
    });

    // ── Bulk "Mark remaining" ────────────────────────────────────────

    it("calls onMarkRemainingAsPresent with unmarked student IDs", async () => {
        const onMarkRemaining = vi.fn();
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
            createMockStudent({ student_id: "stu-2", status: null }),
            createMockStudent({ student_id: "stu-3", status: null }),
        ];
        render(
            <CbcAttendanceGrid
                {...defaultProps}
                students={students}
                onMarkRemainingAsPresent={onMarkRemaining}
            />
        );
        const markBtn = screen.getByText("Mark remaining as Present");
        await userEvent.click(markBtn);
        expect(onMarkRemaining).toHaveBeenCalledWith(["stu-2", "stu-3"]);
    });

    // ── Optimistic queue ─────────────────────────────────────────────

    it("displays pending statuses from localQueue", () => {
        const localQueue: OfflineAttendanceEntry[] = [
            {
                localId: "local-1",
                periodId: "period-1",
                studentId: "stu-1",
                status: "ABSENT",
                timestamp: Date.now(),
                retryCount: 0,
            },
        ];
        render(<CbcAttendanceGrid {...defaultProps} localQueue={localQueue} />);
        // The student row should show syncPending when in the queue
        const syncPendings = screen.getAllByTestId("sync-pending");
        expect(syncPendings.length).toBeGreaterThanOrEqual(1);
    });

    // ── Read-only mode ───────────────────────────────────────────────

    it("disables editing when canEdit is false and records user", () => {
        render(
            <CbcAttendanceGrid {...defaultProps} canEdit={false} recordedByUserId="teacher-2" />
        );
        expect(screen.getByText("View-only mode")).toBeTruthy();
        // Student rows should have readOnly prop
        const readOnlyIndicators = screen.getAllByTestId("read-only");
        expect(readOnlyIndicators.length).toBe(3);
    });

    it("hides 'Mark remaining' button when read-only", () => {
        const students = [
            createMockStudent({ student_id: "stu-1", status: "PRESENT", log_id: "log-1" }),
            createMockStudent({ student_id: "stu-2", status: null }),
        ];
        render(
            <CbcAttendanceGrid
                {...defaultProps}
                students={students}
                canEdit={false}
                recordedByUserId="teacher-2"
            />
        );
        expect(screen.queryByText("Mark remaining as Present")).toBeNull();
    });

    // ── Saving indicator ─────────────────────────────────────────────

    it("shows saving indicator for students in savingStudentIds", () => {
        const savingStudentIds = new Set<string>(["stu-1"]);
        render(<CbcAttendanceGrid {...defaultProps} savingStudentIds={savingStudentIds} />);
        const savingIndicators = screen.getAllByTestId("is-saving");
        expect(savingIndicators.length).toBe(1);
    });

    // ── Edge cases ───────────────────────────────────────────────────

    it("handles single student", () => {
        render(<CbcAttendanceGrid {...defaultProps} students={[createMockStudent()]} />);
        expect(screen.getByText("1 student")).toBeTruthy();
        const rows = screen.getAllByTestId("student-row");
        expect(rows).toHaveLength(1);
    });

    it("handles null periodId with no students", () => {
        render(<CbcAttendanceGrid {...defaultProps} students={[]} periodId={null} />);
        // Falls through to empty state
        expect(
            screen.getByText("No students enrolled in this class for the current term.")
        ).toBeTruthy();
    });
});
