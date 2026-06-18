/**
 * Tests for CBC Attendance API client.
 *
 * Mocks the underlying `api` client to verify request URLs, method,
 * and correct deserialization shapes.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import * as attendanceApi from "@/features/cbc/api/attendance";
import type { AttendanceStatus } from "@/features/cbc/types";

// ─── Mock the api client ──────────────────────────────────────────────────

vi.mock("@/lib/api/client", () => ({
    api: {
        get: vi.fn(),
        post: vi.fn(),
        put: vi.fn(),
        patch: vi.fn(),
        delete: vi.fn(),
    },
}));

const { api } = await import("@/lib/api/client");
const mockGet = vi.mocked(api.get);
const mockPost = vi.mocked(api.post);

beforeEach(() => {
    vi.clearAllMocks();
});

// ═══════════════════════════════════════════════════════════════════════════
// CATEGORY 1: Period queries
// ═══════════════════════════════════════════════════════════════════════════

describe("fetchCbcAttendancePeriods", () => {
    it("GETs periods for a class on a given date", async () => {
        mockGet.mockResolvedValueOnce([]);
        const result = await attendanceApi.fetchCbcAttendancePeriods("class-1", "2026-06-18");
        expect(mockGet).toHaveBeenCalledWith(
            "/api/v1/cbc/classes/class-1/attendance/periods?date=2026-06-18"
        );
        expect(result).toEqual([]);
    });

    it("returns typed period data", async () => {
        mockGet.mockResolvedValueOnce([
            {
                id: "period-1",
                tenant_id: "t1",
                school_id: "s1",
                academic_term_id: "term-1",
                class_id: "class-1",
                cbc_learning_area_id: "area-1",
                date_recorded: "2026-06-18",
            },
        ]);
        const periods = await attendanceApi.fetchCbcAttendancePeriods("class-1", "2026-06-18");
        expect(periods).toHaveLength(1);
        expect(periods[0].id).toBe("period-1");
        expect(periods[0].cbc_learning_area_id).toBe("area-1");
    });
});

describe("fetchCbcAttendancePeriodSummaries", () => {
    it("GETs period summaries for a date range", async () => {
        mockGet.mockResolvedValueOnce([]);
        await attendanceApi.fetchCbcAttendancePeriodSummaries(
            "class-1",
            "2026-06-01",
            "2026-06-30"
        );
        expect(mockGet).toHaveBeenCalledWith(
            "/api/v1/cbc/classes/class-1/attendance/periods?from=2026-06-01&to=2026-06-30"
        );
    });

    it("returns enriched summary data", async () => {
        mockGet.mockResolvedValueOnce([
            {
                id: "period-1",
                date_recorded: "2026-06-18",
                cbc_learning_area_id: "area-1",
                learning_area_name: "Mathematics",
                recorded_by_name: "John Otieno",
                recorded_by_id: "teacher-1",
                recorded_at: "2026-06-18T08:00:00Z",
                total_students: 3,
                present_count: 2,
                absent_count: 1,
                late_count: 0,
                excused_count: 0,
                unmarked_count: 0,
            },
        ]);
        const summaries = await attendanceApi.fetchCbcAttendancePeriodSummaries(
            "class-1",
            "2026-06-01",
            "2026-06-30"
        );
        expect(summaries[0].total_students).toBe(3);
        expect(summaries[0].present_count).toBe(2);
        expect(summaries[0].learning_area_name).toBe("Mathematics");
    });
});

describe("fetchCbcAttendancePeriodDetail", () => {
    it("GETs a single period summary by ID", async () => {
        mockGet.mockResolvedValueOnce({
            id: "period-1",
            date_recorded: "2026-06-18",
            cbc_learning_area_id: "area-1",
            learning_area_name: "Math",
            recorded_by_name: "John Otieno",
            recorded_by_id: "teacher-1",
            recorded_at: "2026-06-18T08:00:00Z",
            total_students: 3,
            present_count: 2,
            absent_count: 1,
            late_count: 0,
            excused_count: 0,
            unmarked_count: 0,
        });
        const result = await attendanceApi.fetchCbcAttendancePeriodDetail("period-1");
        expect(mockGet).toHaveBeenCalledWith("/api/v1/cbc/attendance/periods/period-1");
        expect(result.total_students).toBe(3);
    });
});

// ═══════════════════════════════════════════════════════════════════════════
// CATEGORY 2: Period mutations
// ═══════════════════════════════════════════════════════════════════════════

describe("createCbcAttendancePeriod", () => {
    it("POSTs to create a new attendance period", async () => {
        mockPost.mockResolvedValueOnce({
            id: "period-1",
            tenant_id: "t1",
            school_id: "s1",
            academic_term_id: "term-1",
            class_id: "class-1",
            cbc_learning_area_id: "area-1",
            date_recorded: "2026-06-18",
        });
        const result = await attendanceApi.createCbcAttendancePeriod(
            "class-1",
            "area-1",
            "2026-06-18"
        );
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/classes/class-1/attendance/periods", {
            cbc_learning_area_id: "area-1",
            date_recorded: "2026-06-18",
        });
        expect(result.cbc_learning_area_id).toBe("area-1");
        expect(result.date_recorded).toBe("2026-06-18");
    });

    it("sends snake_case payload fields", async () => {
        mockPost.mockResolvedValueOnce({} as unknown as Record<string, unknown>);
        await attendanceApi.createCbcAttendancePeriod("class-1", "area-1", "2026-06-18");
        const callArgs = mockPost.mock.calls[0][1];
        expect(callArgs).toHaveProperty("cbc_learning_area_id");
        expect(callArgs).toHaveProperty("date_recorded");
    });
});

// ═══════════════════════════════════════════════════════════════════════════
// CATEGORY 3: Log queries
// ═══════════════════════════════════════════════════════════════════════════

describe("fetchCbcAttendanceLogs", () => {
    it("GETs logs for a period with recorder details", async () => {
        mockGet.mockResolvedValueOnce([
            {
                id: "log-1",
                tenant_id: "t1",
                cbc_attendance_period_id: "period-1",
                student_id: "stu-1",
                status: "PRESENT",
                remarks: null,
                recorded_by: "teacher-1",
                recorder_first_name: "John",
                recorder_last_name: "Otieno",
                recorded_by_label: "John Otieno",
            },
        ]);
        const logs = await attendanceApi.fetchCbcAttendanceLogs("period-1");
        expect(mockGet).toHaveBeenCalledWith("/api/v1/cbc/attendance/periods/period-1/logs");
        expect(logs).toHaveLength(1);
        expect(logs[0].recorded_by_label).toBe("John Otieno");
        expect(logs[0].status).toBe("PRESENT");
    });
});

// ═══════════════════════════════════════════════════════════════════════════
// CATEGORY 4: Log mutations
// ═══════════════════════════════════════════════════════════════════════════

describe("saveAttendanceMark", () => {
    it("POSTs a single attendance mark", async () => {
        mockPost.mockResolvedValueOnce({
            id: "log-1",
            tenant_id: "t1",
            cbc_attendance_period_id: "period-1",
            student_id: "stu-1",
            status: "PRESENT",
            remarks: null,
            recorded_by: "teacher-1",
        });
        const result = await attendanceApi.saveAttendanceMark("period-1", "stu-1", "PRESENT");
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/attendance/logs", {
            cbc_attendance_period_id: "period-1",
            student_id: "stu-1",
            status: "PRESENT",
            remarks: null,
        });
        expect(result.status).toBe("PRESENT");
    });

    it("sends ABSENT status correctly", async () => {
        mockPost.mockResolvedValueOnce({} as unknown as Record<string, unknown>);
        await attendanceApi.saveAttendanceMark("period-1", "stu-2", "ABSENT", "Sick");
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/attendance/logs", {
            cbc_attendance_period_id: "period-1",
            student_id: "stu-2",
            status: "ABSENT",
            remarks: "Sick",
        });
    });

    it("sends LATE status correctly", async () => {
        mockPost.mockResolvedValueOnce({} as unknown as Record<string, unknown>);
        await attendanceApi.saveAttendanceMark("period-1", "stu-3", "LATE");
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/attendance/logs", {
            cbc_attendance_period_id: "period-1",
            student_id: "stu-3",
            status: "LATE",
            remarks: null,
        });
    });

    it("sends EXCUSED status correctly", async () => {
        mockPost.mockResolvedValueOnce({} as unknown as Record<string, unknown>);
        await attendanceApi.saveAttendanceMark("period-1", "stu-4", "EXCUSED");
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/attendance/logs", {
            cbc_attendance_period_id: "period-1",
            student_id: "stu-4",
            status: "EXCUSED",
            remarks: null,
        });
    });
});

describe("saveAttendanceBatch", () => {
    it("POSTs multiple marks at once", async () => {
        mockPost.mockResolvedValueOnce([]);
        const marks = [
            { student_id: "stu-1", status: "PRESENT" as AttendanceStatus },
            { student_id: "stu-2", status: "ABSENT" as AttendanceStatus, remarks: "Sick" },
        ];
        await attendanceApi.saveAttendanceBatch("period-1", marks);
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/attendance/logs/batch", {
            cbc_attendance_period_id: "period-1",
            marks,
        });
    });

    it("sends the full marks array", async () => {
        mockPost.mockResolvedValueOnce([]);
        const marks = [{ student_id: "stu-1", status: "PRESENT" as AttendanceStatus }];
        await attendanceApi.saveAttendanceBatch("period-1", marks);
        const body = mockPost.mock.calls[0][1] as { marks: unknown[] };
        expect(body.marks).toHaveLength(1);
    });
});

describe("markRemainingAsPresent", () => {
    it("POSTs unmarked student IDs as PRESENT in batch", async () => {
        mockPost.mockResolvedValueOnce([]);
        await attendanceApi.markRemainingAsPresent("period-1", ["stu-3", "stu-4"]);
        expect(mockPost).toHaveBeenCalledWith("/api/v1/cbc/attendance/logs/batch", {
            cbc_attendance_period_id: "period-1",
            marks: [
                { student_id: "stu-3", status: "PRESENT" },
                { student_id: "stu-4", status: "PRESENT" },
            ],
        });
    });
});

// ═══════════════════════════════════════════════════════════════════════════
// CATEGORY 5: Analytics queries
// ═══════════════════════════════════════════════════════════════════════════

describe("fetchCbcAttendanceHeatmap", () => {
    it("GETs heatmap data for a class and term", async () => {
        mockGet.mockResolvedValueOnce([
            { date: "2026-06-15", period_count: 1, present_rate: 100.0, total_marks: 3 },
            { date: "2026-06-16", period_count: 1, present_rate: 66.67, total_marks: 3 },
        ]);
        const result = await attendanceApi.fetchCbcAttendanceHeatmap("class-1", "term-1");
        expect(mockGet).toHaveBeenCalledWith(
            "/api/v1/cbc/classes/class-1/attendance/heatmap?term_id=term-1"
        );
        expect(result).toHaveLength(2);
        expect(result[0].present_rate).toBe(100.0);
    });
});

describe("fetchCbcAttendanceGaps", () => {
    it("GETs gaps for a date range", async () => {
        mockGet.mockResolvedValueOnce([
            {
                slot_id: "slot-1",
                class_id: "class-1",
                cbc_learning_area_id: "area-1",
                learning_area_name: "Mathematics",
                day_of_week: 1,
                start_time: "08:00",
                end_time: "08:40",
                date: "2026-06-17",
            },
        ]);
        const result = await attendanceApi.fetchCbcAttendanceGaps(
            "class-1",
            "2026-06-15",
            "2026-06-19"
        );
        expect(mockGet).toHaveBeenCalledWith(
            "/api/v1/cbc/classes/class-1/attendance/gaps?from=2026-06-15&to=2026-06-19"
        );
        expect(result).toHaveLength(1);
        expect(result[0].learning_area_name).toBe("Mathematics");
    });
});

// ═══════════════════════════════════════════════════════════════════════════
// CATEGORY 6: Dashboard queries
// ═══════════════════════════════════════════════════════════════════════════

describe("fetchTeacherTodaySlots", () => {
    it("GETs today's slots for a teacher", async () => {
        mockGet.mockResolvedValueOnce([
            {
                slot_id: "slot-1",
                class_id: "class-1",
                class_name: "Grade 7 East",
                learning_area_name: "Mathematics",
                learning_area_id: "area-1",
                start_time: "08:00",
                end_time: "08:40",
                day_of_week: 1,
                attendance_period_id: "period-1",
                status: "done",
                total_students: 3,
                marked_count: 3,
                is_usual_teacher: true,
                academic_term_id: "term-1",
            },
        ]);
        const result = await attendanceApi.fetchTeacherTodaySlots("teacher-1");
        expect(mockGet).toHaveBeenCalledWith(
            "/api/v1/cbc/attendance/slots/today?teacher_id=teacher-1"
        );
        expect(result).toHaveLength(1);
        expect(result[0].status).toBe("done");
    });
});

describe("fetchClassStudents", () => {
    it("GETs enrolled students for a class", async () => {
        mockGet.mockResolvedValueOnce([
            {
                student_id: "stu-1",
                student_name: "Alice Kimani",
                first_name: "Alice",
                last_name: "Kimani",
                gender: "F",
            },
        ]);
        const result = await attendanceApi.fetchClassStudents("class-1", "term-1");
        expect(mockGet).toHaveBeenCalledWith(
            "/api/v1/cbc/classes/class-1/students?academic_term_id=term-1"
        );
        expect(result).toHaveLength(1);
        expect(result[0].student_name).toBe("Alice Kimani");
    });
});
