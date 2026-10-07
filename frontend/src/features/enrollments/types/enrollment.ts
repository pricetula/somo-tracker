export type EnrollmentStatus = "ACTIVE" | "PROMOTED" | "REPEATING" | "GRADUATED";

export interface Enrollment {
    id: string;
    student_id: string;
    class_room_id: string;
    academic_year_id: string;
    academic_term_id?: string;
    enrolled_at: string;
    status: EnrollmentStatus;
    class_name?: string;
    grade?: string;
    stream?: string;
    student_name?: string;
    admission_number?: string;
}

export interface CreateEnrollmentRequest {
    student_id: string;
    enrollment_date: string;
    metadata?: Record<string, unknown>;
}
