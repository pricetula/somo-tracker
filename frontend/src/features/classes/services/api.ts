import type { ClassListItem, ClassDetail } from "../types/class";
import { api } from "@/lib/api/client";

export async function listClasses(params: {
    page?: number;
    limit?: number;
    search?: string;
    filters?: Record<string, string | string[]>;
}): Promise<{ items: ClassListItem[]; total: number }> {
    const page = params.page ?? 1;
    const limit = params.limit ?? 50;
    const search = params.search ?? "";
    const grades = params.filters?.["grade-filter"] ?? params.filters?.grade;
    const streams = params.filters?.["stream-filter"] ?? params.filters?.stream;

    const query = new URLSearchParams({
        page: String(page),
        limit: String(limit),
    });
    if (search) query.set("search", search);
    if (Array.isArray(grades) && grades.length > 0) {
        query.set("grade", grades.join(","));
    } else if (typeof grades === "string" && grades.trim() !== "") {
        query.set("grade", grades);
    }
    if (Array.isArray(streams) && streams.length > 0) {
        query.set("stream", streams.join(","));
    } else if (typeof streams === "string" && streams.trim() !== "") {
        query.set("stream", streams);
    }

    const res = await api.get<{ items: ClassListItem[]; total: number }>(
        `/api/school/classes?${query.toString()}`
    );
    return res;
}

export async function getClass(id: string): Promise<ClassDetail | null> {
    const res = await api.get<ClassDetail>(`/api/school/classes/${id}`);
    return res ?? null;
}

export async function createClass(data: {
    name: string;
    gradeId: string;
    streamId?: string;
}): Promise<ClassListItem> {
    const res = await api.post<ClassListItem>(`/api/school/classes`, data);
    return res;
}
