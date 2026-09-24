import { api } from "./client";

export type StudentListItem = {
    student_id: string;
    admission_number: string;
    full_name: string;
    date_of_birth: string;
    gender: string;
    class_id?: string;
    class_name?: string;
};

export type StudentListResponse = {
    items: StudentListItem[];
    total: number;
};

export async function listStudents(params?: {
    page?: number;
    limit?: number;
    search?: string;
    class_id?: string;
}): Promise<StudentListResponse> {
    const qs = new URLSearchParams();
    if (params?.page) qs.set("page", String(params.page));
    if (params?.limit) qs.set("limit", String(params.limit));
    if (params?.search) qs.set("search", params.search);
    if (params?.class_id) qs.set("class_id", params.class_id);
    const url = `/api/students${qs.toString() ? `?${qs}` : ""}`;
    return api.get<StudentListResponse>(url);
}

export interface DeleteStudentsResponse {
    code: string;
    message: string;
    errors: Record<string, unknown>;
}

export async function deleteStudents(studentIds: string[]): Promise<DeleteStudentsResponse> {
    return api.delete<DeleteStudentsResponse>("/api/students", { student_ids: studentIds });
}

export type StudentSummary = {
    total_students: number;
    male_count: number;
    female_count: number;
    unassigned_count: number;
    unlinked_guardians_count: number;
};

export async function getStudentSummary(): Promise<StudentSummary> {
    return api.get<StudentSummary>("/api/students/summary");
}
