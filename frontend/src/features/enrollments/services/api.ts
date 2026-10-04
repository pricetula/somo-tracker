import { api } from "@/lib/api/client";
import type { Enrollment, CreateEnrollmentRequest } from "../types/enrollment";

export type UnassignedStudent = {
    student_id: string;
    full_name: string;
    admission_number: string;
};

export async function listByClass(
    classId: string,
    params?: { page?: number; limit?: number }
): Promise<{ items: Enrollment[]; total: number }> {
    const page = params?.page ?? 1;
    const limit = params?.limit ?? 50;
    return api.get<{ items: Enrollment[]; total: number }>(
        `/api/classes/${classId}/enrollments?page=${page}&limit=${limit}`
    );
}

export async function listUnassigned(params?: {
    page?: number;
    limit?: number;
}): Promise<{ items: UnassignedStudent[]; total: number }> {
    const page = params?.page ?? 1;
    const limit = params?.limit ?? 50;
    return api.get<{ items: UnassignedStudent[]; total: number }>(
        `/api/students/unassigned?page=${page}&limit=${limit}`
    );
}

export async function createBatch(
    classId: string,
    data: CreateEnrollmentRequest[]
): Promise<{ items: Enrollment[] }> {
    return api.post<{ items: Enrollment[] }>(`/api/classes/${classId}/enrollments`, data);
}

export async function updateEnrollment(
    id: string,
    data: Partial<CreateEnrollmentRequest>
): Promise<Enrollment> {
    return api.patch<Enrollment>(`/api/enrollments/${id}`, data);
}

export async function deleteEnrollment(id: string): Promise<void> {
    return api.delete<void>(`/api/enrollments/${id}`);
}
