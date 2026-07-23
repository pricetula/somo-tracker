/**
 * Class Teachers API functions.
 *
 * Endpoints (from backend/internal/classteachers/handler.go):
 *   POST   /api/v1/class-teachers               — assign teacher to class
 *   GET    /api/v1/class-teachers/:id             — get assignment
 *   GET    /api/v1/class-teachers/by-class/:classId — list by class
 *   GET    /api/v1/class-teachers/by-teacher/:userId — list by teacher
 *   DELETE /api/v1/class-teachers/:id             — remove assignment
 */

import { api } from "./client";

// ─── Domain Types ─────────────────────────────────────────────────────────

export interface ClassTeacher {
    id: string;
    class_id: string;
    user_id: string;
    teacher_name?: string;
    learning_area_id?: string | null;
    learning_area?: string | null;
    teacher_role: string;
    created_at: string;
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface ClassTeacherListResponse {
    items: ClassTeacher[];
    total: number;
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface CreateClassTeacherPayload {
    user_id: string;
    class_id: string;
    learning_area_id?: string | null;
    teacher_role: string;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** Assign a teacher to a class. */
export async function createClassTeacher(data: CreateClassTeacherPayload): Promise<ClassTeacher> {
    return api.post<ClassTeacher>("/api/v1/class-teachers", data);
}

/** Get a single class teacher assignment by ID. */
export async function getClassTeacher(id: string): Promise<ClassTeacher> {
    return api.get<ClassTeacher>(`/api/v1/class-teachers/${id}`);
}

/** List all teacher assignments for a given class. */
export async function listClassTeachersByClass(classId: string): Promise<ClassTeacherListResponse> {
    return api.get<ClassTeacherListResponse>(`/api/v1/class-teachers/by-class/${classId}`);
}

/** List all class assignments for a given teacher. */
export async function listClassTeachersByTeacher(
    userId: string
): Promise<ClassTeacherListResponse> {
    return api.get<ClassTeacherListResponse>(`/api/v1/class-teachers/by-teacher/${userId}`);
}

/** Remove a class teacher assignment. */
export async function deleteClassTeacher(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/class-teachers`, { id });
}
