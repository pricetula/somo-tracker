/**
 * CBC Timetable API client.
 *
 * Endpoints (to be implemented in the Go backend):
 *   GET    /api/v1/cbc/classes/:classId/timetable          → CbcTimetableSlot[]
 *   POST   /api/v1/cbc/classes/:classId/timetable           → CbcTimetableSlot
 *   PUT    /api/v1/cbc/timetable/:slotId                    → CbcTimetableSlot
 *   DELETE /api/v1/cbc/timetable/:slotId                    → void
 *   GET    /api/v1/cbc/timetable/:slotId/conflicts          → CbcSlotConflict[]
 *   POST   /api/v1/cbc/timetable/duplicate-day              → BulkOperationResult
 *   POST   /api/v1/cbc/timetable/copy-from-class            → BulkOperationResult
 *   GET    /api/v1/cbc/learning-areas?grade_id=:gradeId     → LearningAreaOption[]
 *   GET    /api/v1/cbc/teachers?school_id=:schoolId         → TeacherOption[]
 *   GET    /api/v1/cbc/classes/:classId/teachers            → TeacherOption[] (scoped)
 *   GET    /api/v1/cbc/room-autocomplete?q=:query           → string[]
 */

import { api } from "@/lib/api/client";
import type {
    CbcTimetableSlot,
    CbcTimetableSlotCreatePayload,
    CbcTimetableSlotUpdatePayload,
    CbcSlotConflict,
    LearningAreaOption,
    TeacherOption,
    DuplicateDayPayload,
    CopyFromClassPayload,
    BulkOperationResult,
    OperatingDay,
} from "@/features/cbc/types";

// ─── Fetch slots for a class ──────────────────────────────────────────────

export async function fetchCbcTimetableSlots(classId: string): Promise<CbcTimetableSlot[]> {
    return api.get<CbcTimetableSlot[]>(`/api/v1/cbc/classes/${classId}/timetable`);
}

// ─── Create a new slot ────────────────────────────────────────────────────

export async function createCbcTimetableSlot(
    classId: string,
    payload: CbcTimetableSlotCreatePayload
): Promise<CbcTimetableSlot> {
    return api.post<CbcTimetableSlot>(`/api/v1/cbc/classes/${classId}/timetable`, payload);
}

// ─── Update an existing slot ──────────────────────────────────────────────

export async function updateCbcTimetableSlot(
    payload: CbcTimetableSlotUpdatePayload
): Promise<CbcTimetableSlot> {
    const { id, ...data } = payload;
    return api.put<CbcTimetableSlot>(`/api/v1/cbc/timetable/${id}`, data);
}

// ─── Delete a slot ────────────────────────────────────────────────────────

export async function deleteCbcTimetableSlot(slotId: string): Promise<void> {
    return api.delete(`/api/v1/cbc/timetable/${slotId}`);
}

// ─── Check how many attendance periods are linked to a slot ──────────────

export async function fetchSlotAttendanceCount(slotId: string): Promise<{ count: number }> {
    return api.get<{ count: number }>(`/api/v1/cbc/timetable/${slotId}/attendance-count`);
}

// ─── Conflict pre-check ──────────────────────────────────────────────────

/**
 * Best-effort pre-check for teacher/room conflicts while the side panel is open.
 * Not authoritative — the DB exclusion constraints are the source of truth.
 */
export async function checkSlotConflicts(
    slotId: string | null, // null = new slot (no id yet)
    teacherId: string,
    dayOfWeek: number,
    startTime: string,
    endTime: string,
    academicYearId: string,
    schoolId: string,
    roomIdentifier: string | null,
    excludeClassId?: string // exclude own class for edit-in-place
): Promise<CbcSlotConflict[]> {
    const params = new URLSearchParams({
        teacher_id: teacherId,
        day_of_week: String(dayOfWeek),
        start_time: startTime,
        end_time: endTime,
        academic_year_id: academicYearId,
        school_id: schoolId,
    });
    if (roomIdentifier) params.set("room_identifier", roomIdentifier);
    if (slotId) params.set("exclude_slot_id", slotId);
    if (excludeClassId) params.set("exclude_class_id", excludeClassId);

    return api.get<CbcSlotConflict[]>(`/api/v1/cbc/timetable/conflicts?${params.toString()}`);
}

// ─── Duplicate day ───────────────────────────────────────────────────────

export async function duplicateDay(payload: DuplicateDayPayload): Promise<BulkOperationResult> {
    return api.post<BulkOperationResult>("/api/v1/cbc/timetable/duplicate-day", payload);
}

// ─── Copy from another class ─────────────────────────────────────────────

export async function copyTimetableFromClass(
    payload: CopyFromClassPayload
): Promise<BulkOperationResult> {
    return api.post<BulkOperationResult>("/api/v1/cbc/timetable/copy-from-class", payload);
}

// ─── Learning areas for a grade ──────────────────────────────────────────

export async function fetchCbcLearningAreas(gradeId: string): Promise<LearningAreaOption[]> {
    return api.get<LearningAreaOption[]>(`/api/v1/cbc/learning-areas?grade_id=${gradeId}`);
}

// ─── Teachers for a school ───────────────────────────────────────────────

export async function fetchCbcTeachers(schoolId: string): Promise<TeacherOption[]> {
    return api.get<TeacherOption[]>(`/api/v1/cbc/teachers?school_id=${schoolId}`);
}

// ─── Teachers scoped to a class + learning area ──────────────────────────

export async function fetchCbcClassTeachers(
    classId: string,
    learningAreaId?: string
): Promise<TeacherOption[]> {
    const params = new URLSearchParams({ class_id: classId });
    if (learningAreaId) params.set("learning_area_id", learningAreaId);
    return api.get<TeacherOption[]>(`/api/v1/cbc/class-teachers?${params.toString()}`);
}

// ─── Room autocomplete ───────────────────────────────────────────────────

export async function fetchRoomAutocomplete(query: string): Promise<string[]> {
    return api.get<string[]>(`/api/v1/cbc/room-autocomplete?q=${encodeURIComponent(query)}`);
}

// ─── Operating days for a school ─────────────────────────────────────────

export async function fetchOperatingDays(schoolId: string): Promise<OperatingDay[]> {
    return api.get<OperatingDay[]>(`/api/v1/cbc/operating-days?school_id=${schoolId}`);
}
