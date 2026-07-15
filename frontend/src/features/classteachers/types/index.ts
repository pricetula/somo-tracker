/**
 * TypeScript interfaces for the Class Teachers feature.
 *
 * Maps to backend/internal/classteachers/domain.go
 */

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

export interface ClassTeacherListResponse {
    items: ClassTeacher[];
    total: number;
}

export interface CreateClassTeacherPayload {
    user_id: string;
    class_id: string;
    learning_area_id?: string | null;
    teacher_role: string;
}
