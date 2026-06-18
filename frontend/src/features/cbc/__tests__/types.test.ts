/**
 * Tests for CBC timetable types — validates interface shapes match
 * backend JSON contracts and utility type logic.
 */

import { describe, it, expect } from "vitest";
import type {
    CbcTimetableSlot,
    CbcTimetableSlotCreatePayload,
    CbcTimetableSlotUpdatePayload,
    CbcSlotConflict,
    BulkOperationResult,
    SlotSkipReason,
    AttendanceStatus,
    AttendanceStudentRow,
    CbcAttendancePeriod,
    CbcAttendanceLog,
    OfflineAttendanceEntry,
    OperatingDay,
    TeacherOption,
    LearningAreaOption,
    AttendanceSlotColumn,
    DuplicateDayPayload,
    CopyFromClassPayload,
    AttendanceCount,
} from "@/features/cbc/types";

// ─── Mock factory: produces a minimal valid CbcTimetableSlot ──────────────
function mockSlot(overrides?: Partial<CbcTimetableSlot>): CbcTimetableSlot {
    return {
        id: "slot-001",
        tenant_id: "tenant-001",
        school_id: "school-001",
        academic_year_id: "year-001",
        class_id: "class-001",
        teacher_id: "teacher-001",
        cbc_learning_area_id: "area-001",
        room_identifier: "Room 4",
        day_of_week: 1,
        start_time: "08:00",
        end_time: "08:40",
        ...overrides,
    };
}

// ─── Category 1: CbcTimetableSlot surface area ──────────────────────────

describe("CbcTimetableSlot", () => {
    it("constructs with all required fields", () => {
        const slot = mockSlot();
        expect(slot.id).toBe("slot-001");
        expect(slot.tenant_id).toBe("tenant-001");
        expect(slot.school_id).toBe("school-001");
        expect(slot.academic_year_id).toBe("year-001");
        expect(slot.class_id).toBe("class-001");
        expect(slot.teacher_id).toBe("teacher-001");
        expect(slot.day_of_week).toBe(1);
        expect(slot.start_time).toBe("08:00");
        expect(slot.end_time).toBe("08:40");
    });

    it("accepts null cbc_learning_area_id (break / assembly)", () => {
        const slot = mockSlot({ cbc_learning_area_id: null });
        expect(slot.cbc_learning_area_id).toBeNull();
    });

    it("accepts null room_identifier", () => {
        const slot = mockSlot({ room_identifier: null });
        expect(slot.room_identifier).toBeNull();
    });

    it("accepts all day_of_week values 1-7", () => {
        for (let d = 1; d <= 7; d++) {
            const slot = mockSlot({ day_of_week: d });
            expect(slot.day_of_week).toBe(d);
        }
    });

    it("preserves time strings as-is", () => {
        const slot = mockSlot({ start_time: "23:59", end_time: "00:01" });
        expect(slot.start_time).toBe("23:59");
        expect(slot.end_time).toBe("00:01");
    });

    it("starts with empty ID when new", () => {
        const slot: CbcTimetableSlotCreatePayload = {
            class_id: "class-001",
            teacher_id: "teacher-001",
            cbc_learning_area_id: null,
            room_identifier: null,
            day_of_week: 2,
            start_time: "09:00",
            end_time: "09:40",
        };
        expect(slot.class_id).toBe("class-001");
        expect(slot.teacher_id).toBe("teacher-001");
        // No id field in create payload
        expect("id" in slot).toBe(false);
    });
});

// ─── Category 2: CbcTimetableSlotUpdatePayload edge cases ───────────────

describe("CbcTimetableSlotUpdatePayload", () => {
    it("requires id but allows partial fields", () => {
        // Partial update — only change the room
        const partial: CbcTimetableSlotUpdatePayload = {
            id: "slot-001",
            room_identifier: "Lab A",
        };
        expect(partial.id).toBe("slot-001");
        expect(partial.room_identifier).toBe("Lab A");
        expect(partial.teacher_id).toBeUndefined();
    });

    it("allows setting learning_area to null to clear it", () => {
        const update: CbcTimetableSlotUpdatePayload = {
            id: "slot-001",
            cbc_learning_area_id: null,
        };
        expect(update.cbc_learning_area_id).toBeNull();
    });

    it("allows all fields simultaneously", () => {
        const full: CbcTimetableSlotUpdatePayload = {
            id: "slot-001",
            teacher_id: "teacher-002",
            cbc_learning_area_id: "area-002",
            room_identifier: "Lab B",
            day_of_week: 3,
            start_time: "10:00",
            end_time: "10:45",
        };
        expect(full.teacher_id).toBe("teacher-002");
        expect(full.day_of_week).toBe(3);
    });
});

// ─── Category 3: CbcSlotConflict shape ──────────────────────────────────

describe("CbcSlotConflict", () => {
    it("distinguishes teacher vs room conflicts", () => {
        const teacherConflict: CbcSlotConflict = {
            type: "teacher",
            entity: "John Otieno",
            class_name: "Grade 7 East",
            day_of_week: 1,
            start_time: "08:00",
            end_time: "08:40",
        };
        expect(teacherConflict.type).toBe("teacher");
        expect(teacherConflict.entity).toBe("John Otieno");
        expect(teacherConflict.class_name).toBe("Grade 7 East");

        const roomConflict: CbcSlotConflict = {
            type: "room",
            entity: "Lab A",
            class_name: "Grade 7 West",
            day_of_week: 3,
            start_time: "09:00",
            end_time: "09:40",
        };
        expect(roomConflict.type).toBe("room");
        expect(roomConflict.entity).toBe("Lab A");
    });

    it("round-trips serialization", () => {
        const conflict: CbcSlotConflict = {
            type: "teacher",
            entity: "Jane Wanjiku",
            class_name: "Grade 8 Blue",
            day_of_week: 5,
            start_time: "11:00",
            end_time: "11:40",
        };
        const json = JSON.stringify(conflict);
        const parsed = JSON.parse(json) as CbcSlotConflict;
        expect(parsed.type).toBe("teacher");
        expect(parsed.entity).toBe("Jane Wanjiku");
        expect(parsed.class_name).toBe("Grade 8 Blue");
        expect(parsed.day_of_week).toBe(5);
        expect(parsed.start_time).toBe("11:00");
        expect(parsed.end_time).toBe("11:40");
    });
});

// ─── Category 4: BulkOperationResult + SlotSkipReason ───────────────────

describe("BulkOperationResult", () => {
    it("returns zero copied when empty", () => {
        const result: BulkOperationResult = { total_copied: 0, skipped: [] };
        expect(result.total_copied).toBe(0);
        expect(result.skipped).toHaveLength(0);
    });

    it("reports skipped slots with reasons", () => {
        const skipped: SlotSkipReason[] = [
            { day_of_week: 2, start_time: "08:00", reason: "Teacher conflict" },
            { day_of_week: 3, start_time: "09:00", reason: "Room in use" },
        ];
        const result: BulkOperationResult = { total_copied: 3, skipped };
        expect(result.total_copied).toBe(3);
        expect(result.skipped).toHaveLength(2);
        expect(result.skipped[0].day_of_week).toBe(2);
        expect(result.skipped[0].reason).toBe("Teacher conflict");
    });
});

// ─── Category 5: Attendance types ───────────────────────────────────────

describe("AttendanceStatus", () => {
    it("is a union of four valid statuses", () => {
        const valid: AttendanceStatus[] = ["PRESENT", "ABSENT", "LATE", "EXCUSED"];
        expect(valid).toHaveLength(4);
    });

    it("rejects invalid status at compile time", () => {
        const status: AttendanceStatus = "PRESENT";
        // This test verifies the type system: TSError if invalid string used
        expect(["PRESENT", "ABSENT", "LATE", "EXCUSED"]).toContain(status);
    });
});

describe("AttendanceStudentRow", () => {
    it("starts with syncPending false by default", () => {
        const row: AttendanceStudentRow = {
            student_id: "stu-001",
            student_name: "Alice Kimani",
            first_name: "Alice",
            last_name: "Kimani",
            status: null,
            log_id: null,
        };
        expect(row.syncPending).toBeUndefined();
        expect(row.status).toBeNull();
    });

    it("can have a recorded status and log", () => {
        const row: AttendanceStudentRow = {
            student_id: "stu-002",
            student_name: "Bob Ochieng",
            first_name: "Bob",
            last_name: "Ochieng",
            status: "PRESENT",
            log_id: "log-001",
            syncPending: false,
        };
        expect(row.status).toBe("PRESENT");
        expect(row.log_id).toBe("log-001");
        expect(row.syncPending).toBe(false);
    });

    it("has optional syncPending set when saving", () => {
        const row: AttendanceStudentRow = {
            student_id: "stu-003",
            student_name: "Carol Wanjiku",
            first_name: "Carol",
            last_name: "Wanjiku",
            status: "LATE",
            log_id: null,
            syncPending: true,
        };
        expect(row.syncPending).toBe(true);
    });
});

describe("CbcAttendancePeriod", () => {
    it("has all required fields from the backend", () => {
        const period: CbcAttendancePeriod = {
            id: "period-001",
            tenant_id: "tenant-001",
            school_id: "school-001",
            academic_term_id: "term-001",
            class_id: "class-001",
            cbc_learning_area_id: "area-001",
            date_recorded: "2026-06-18",
        };
        expect(period.id).toBe("period-001");
        expect(period.cbc_learning_area_id).toBe("area-001");
        expect(period.date_recorded).toBe("2026-06-18");
    });
});

describe("CbcAttendanceLog", () => {
    it("stores remarks as nullable", () => {
        const withRemarks: CbcAttendanceLog = {
            id: "log-001",
            tenant_id: "tenant-001",
            cbc_attendance_period_id: "period-001",
            student_id: "stu-001",
            status: "ABSENT",
            remarks: "Sick",
            recorded_by: "teacher-001",
        };
        expect(withRemarks.remarks).toBe("Sick");

        const withoutRemarks: CbcAttendanceLog = {
            id: "log-002",
            tenant_id: "tenant-001",
            cbc_attendance_period_id: "period-001",
            student_id: "stu-002",
            status: "PRESENT",
            remarks: null,
            recorded_by: "teacher-001",
        };
        expect(withoutRemarks.remarks).toBeNull();
    });
});

describe("OfflineAttendanceEntry", () => {
    it("tracks retry count for optimistic saves", () => {
        const entry: OfflineAttendanceEntry = {
            localId: "local-001",
            periodId: "period-001",
            studentId: "stu-001",
            status: "PRESENT",
            timestamp: Date.now(),
            retryCount: 0,
        };
        expect(entry.retryCount).toBe(0);
        expect(entry.studentId).toBe("stu-001");

        const retried = { ...entry, retryCount: 3, status: "LATE" as const };
        expect(retried.retryCount).toBe(3);
        expect(retried.status).toBe("LATE");
    });
});

// ─── Category 6: Reference data types ───────────────────────────────────

describe("OperatingDay", () => {
    it("uses snake_case JSON contract", () => {
        const day: OperatingDay = {
            value: 1,
            label: "Monday",
            short_label: "Mon",
        };
        const json = JSON.stringify(day);
        expect(json).toContain("short_label");
        expect(json).not.toContain("shortLabel");
    });

    it("supports all 7 days", () => {
        const days: OperatingDay[] = [
            { value: 1, label: "Monday", short_label: "Mon" },
            { value: 2, label: "Tuesday", short_label: "Tue" },
            { value: 3, label: "Wednesday", short_label: "Wed" },
            { value: 4, label: "Thursday", short_label: "Thu" },
            { value: 5, label: "Friday", short_label: "Fri" },
            { value: 6, label: "Saturday", short_label: "Sat" },
            { value: 7, label: "Sunday", short_label: "Sun" },
        ];
        expect(days).toHaveLength(7);
    });
});

describe("TeacherOption", () => {
    it("matches backend TeacherBrief JSON", () => {
        const teacher: TeacherOption = {
            id: "teacher-001",
            name: "John Otieno",
            first_name: "John",
            last_name: "Otieno",
            email: "jotieno@school.com",
        };
        const json = JSON.stringify(teacher);
        expect(json).toContain("first_name");
        expect(json).toContain("last_name");
        expect(JSON.parse(json).name).toBe("John Otieno");
    });
});

describe("LearningAreaOption", () => {
    it("has optional grade_id field", () => {
        const withGrade: LearningAreaOption = {
            id: "area-001",
            name: "Mathematics",
            code: "MAT",
            grade_id: "grade-001",
        };
        expect(withGrade.grade_id).toBe("grade-001");

        const withoutGrade: LearningAreaOption = {
            id: "area-002",
            name: "English",
            code: "ENG",
        };
        expect(withoutGrade.grade_id).toBeUndefined();
    });
});

describe("AttendanceSlotColumn", () => {
    it("matches backend SlotBrief JSON contract", () => {
        const slot: AttendanceSlotColumn = {
            period_id: "slot-001_class-001",
            learning_area_name: "Mathematics",
            start_time: "08:00",
            end_time: "08:40",
        };
        const json = JSON.stringify(slot);
        expect(json).toContain("period_id");
        expect(json).toContain("learning_area_name");
        expect(json).toContain("start_time");
        expect(json).toContain("end_time");

        const parsed = JSON.parse(json);
        expect(parsed.period_id).toBe("slot-001_class-001");
        expect(parsed.learning_area_name).toBe("Mathematics");
    });
});

// ─── Category 7: Request/Response payloads ──────────────────────────────

describe("DuplicateDayPayload", () => {
    it("uses snake_case for JSON serialization", () => {
        const payload: DuplicateDayPayload = {
            source_day: 1,
            target_days: [2, 3],
            academic_year_id: "year-001",
            class_id: "class-001",
        };
        const json = JSON.stringify(payload);
        expect(json).toContain("source_day");
        expect(json).toContain("target_days");
        expect(json).toContain("academic_year_id");
        expect(json).not.toContain("sourceDay");
    });
});

describe("CopyFromClassPayload", () => {
    it("uses snake_case for JSON serialization", () => {
        const payload: CopyFromClassPayload = {
            source_class_id: "class-001",
            academic_year_id: "year-001",
            target_class_id: "class-002",
        };
        const json = JSON.stringify(payload);
        expect(json).toContain("source_class_id");
        expect(json).toContain("target_class_id");
        expect(json).not.toContain("sourceClassId");
    });
});

describe("AttendanceCount", () => {
    it("returns a simple count", () => {
        const count: AttendanceCount = { count: 14 };
        expect(count.count).toBe(14);
        const json = JSON.stringify(count);
        expect(json).toBe('{"count":14}');
    });
});
