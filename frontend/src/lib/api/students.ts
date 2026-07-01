/**
 * Students API functions.
 *
 * Endpoints:
 *   GET  /api/v1/students/list       — paginated student listing
 *   POST /api/v1/students            — create student
 *   GET  /api/v1/students/:id        — student detail with enrollments
 *   PUT  /api/v1/students/:id        — update student
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

export interface StudentDetail {
    id: string;
    full_name: string;
    gender: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    class_name?: string | null;
    class_id?: string | null;
    is_active: boolean;
    created_at: string;
    enrollments: Enrollment[];
}

// ─── Response Types ───────────────────────────────────────────────────────

export interface ListStudentsResponse {
    students: Student[];
    total: number;
    page: number;
    limit: number;
}

export interface StudentDetailResponse {
    data: StudentDetail;
}

export interface CreateStudentResponse {
    id: string;
}

export interface CreateEnrollmentResponse {
    id: string;
}

export interface ListEnrollmentsResponse {
    data: Enrollment[];
}

// ─── Payload Types ────────────────────────────────────────────────────────

export interface ListStudentsParams {
    page?: number;
    limit?: number;
    search?: string;
    class_id?: string;
    gender?: string;
}

export interface CreateStudentPayload {
    full_name: string;
    gender?: string;
    date_of_birth?: string | null;
    upi_number?: string | null;
    knec_assessment_number?: string | null;
    class_id?: string | null;
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
    if (params.page) searchParams.set("page", String(params.page));
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.search) searchParams.set("search", params.search);
    if (params.class_id) searchParams.set("class_id", params.class_id);
    if (params.gender) searchParams.set("gender", params.gender);

    const qs = searchParams.toString();
    return api.get<ListStudentsResponse>(`/api/v1/students/list?${qs}`);
}

/** Create a new student. */
export async function createStudent(data: CreateStudentPayload): Promise<CreateStudentResponse> {
    return api.post<CreateStudentResponse>("/api/v1/students", data);
}

/** Get student detail with enrollment history. */
export async function getStudentDetail(id: string): Promise<StudentDetailResponse> {
    return api.get<StudentDetailResponse>(`/api/v1/students/${id}`);
}

/** Update a student. */
export async function updateStudent(id: string, data: UpdateStudentPayload): Promise<void> {
    return api.put<void>(`/api/v1/students/${id}`, data);
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
