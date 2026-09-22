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
