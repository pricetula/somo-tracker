/**
 * Students API functions.
 *
 * Endpoints:
 *   GET  /api/v1/students/list       — paginated student listing
 *   POST /api/v1/students            — batch create students (accepts array)
 *   GET  /api/v1/students/:id        — student detail with enrollments
 *   PUT  /api/v1/students/:id        — update student
 *   DELETE /api/v1/students/:id      — hard-delete student
 *   POST /api/v1/students/:id/enrollments    — create enrollment
 *   GET  /api/v1/students/:id/enrollments    — list enrollments
 */

import { api } from "./client";

// ─── Domain Types ─────────────────────────────────────────────────────────

export interface Student {
    id: string;
    full_name: string;
    gender: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    admission_number?: string | null;
    class_name?: string | null;
    class_id?: string | null;
    is_active: boolean;
    created_at: string;
}

export interface Enrollment {
    id: string;
    student_id: string;
    class_id: string;
    academic_term_id: string;
    term_name: string;
    term_number: number;
    academic_year: string;
    class_name: string;
    status: string;
    created_at: string;
}

export interface BehaviorNoteItem {
    id: string;
    category_name: string;
    description: string;
    date: string;
    status: string;
    is_urgent: boolean;
}

export interface StudentDetail {
    id: string;
    full_name: string;
    gender: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    admission_number?: string | null;
    class_name?: string | null;
    class_id?: string | null;
    is_active: boolean;
    created_at: string;
    enrollments: Enrollment[];
    behavior?: BehaviorNoteItem[];
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface ListStudentsResponse {
    items: Student[];
    total: number;
    page: number;
    limit: number;
}

export interface StudentDetailResponse {
    data: StudentDetail;
}

export interface CreateStudentsResponse {
    ids: string[];
    code: string;
}

export interface CreateEnrollmentResponse {
    id: string;
}

export interface ListEnrollmentsResponse {
    items: Enrollment[];
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface ListStudentsParams {
    page?: number;
    limit?: number;
    search?: string;
    class_id?: string;
    gender?: string;
    enrollment_status?: string;
    /** Filter values keyed by FilterItem id, e.g. { education_level: ["Early_Years"], grade_level: ["G1", "G2"] } */
    filters?: Record<string, string[]>;
}

export interface CreateStudentPayload {
    full_name: string;
    gender?: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    admission_number?: string | null;
    class_id?: string | null;
}

/** Batch create request — wraps an array of students. */
export interface CreateStudentsPayload {
    students: CreateStudentPayload[];
}

export interface UpdateStudentPayload {
    full_name?: string;
    gender?: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    is_active?: boolean;
}

export interface CreateEnrollmentPayload {
    academic_term_id: string;
    class_id: string;
    status?: string;
}

// ─── API Functions ─────────────────────────────────────────────────────────

/** List students with pagination and optional filters. */
export async function listStudents(params: ListStudentsParams = {}): Promise<ListStudentsResponse> {
    const searchParams = new URLSearchParams();

    // Multi-value filters
    const edLevels = params.filters?.education_level ?? [];
    for (const el of edLevels) {
        searchParams.append("education_level", el);
    }
    const grLevels = params.filters?.grade_level ?? [];
    for (const gl of grLevels) {
        searchParams.append("grade_level", gl);
    }

    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    if (params.class_id) searchParams.set("class_id", params.class_id);
    if (params.gender) searchParams.set("gender", params.gender);
    if (params.enrollment_status) searchParams.set("enrollment_status", params.enrollment_status);

    const qs = searchParams.toString();
    return api.get<ListStudentsResponse>(`/api/v1/students/list?${qs}`);
}

/**
 * Create one or more students (batch).
 *
 * Sends `{ "students": [...] }` to POST /api/v1/students.
 * Returns the array of created IDs.
 */
export async function createStudents(data: CreateStudentsPayload): Promise<CreateStudentsResponse> {
    return api.post<CreateStudentsResponse>("/api/v1/students", data);
}

/**
 * Convenience wrapper: create a single student.
 * Sends the student wrapped in a batch and returns the single ID.
 */
export async function createStudent(data: CreateStudentPayload): Promise<{ id: string }> {
    const result = await createStudents({ students: [data] });
    return { id: result.ids[0] };
}

/** Get student detail with enrollment history. */
export async function getStudentDetail(id: string): Promise<StudentDetailResponse> {
    return api.get<StudentDetailResponse>(`/api/v1/students/${id}`);
}

/** Convenience: get student detail, unwrapping the response. */
export async function getStudent(id: string, termId?: string): Promise<StudentDetail> {
    const params = termId ? `?term_id=${termId}` : "";
    const resp = await api.get<StudentDetailResponse>(`/api/v1/students/${id}${params}`);
    return resp.data;
}

/** Update a student. */
export async function updateStudent(id: string, data: UpdateStudentPayload): Promise<void> {
    return api.put<void>(`/api/v1/students/${id}`, data);
}

/** Hard-delete a student. */
export async function deleteStudent(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/students/${id}`);
}

/** Create an enrollment (enroll in class for a term). */
export async function createEnrollment(
    studentId: string,
    data: CreateEnrollmentPayload
): Promise<CreateEnrollmentResponse> {
    return api.post<CreateEnrollmentResponse>(`/api/v1/students/${studentId}/enrollments`, data);
}

/** List enrollments for a student. */
export async function listEnrollments(studentId: string): Promise<ListEnrollmentsResponse> {
    return api.get<ListEnrollmentsResponse>(`/api/v1/students/${studentId}/enrollments`);
}
