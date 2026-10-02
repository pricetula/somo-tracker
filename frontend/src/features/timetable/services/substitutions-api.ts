import { api } from "@/lib/api/client";
import type {
    TimetableSubstitution,
    ListSubstitutionsParams,
    ListSubstitutionsResponse,
    CreateSubstitutionRequest,
    UpdateSubstitutionRequest,
} from "../types/substitution";

interface BackendSubstitution {
    id: string;
    school_id: string;
    class_timetable_slot_id: string;
    substitution_date: string;
    original_teacher_membership_id: string;
    substitute_teacher_membership_id?: string;
    status: string;
    reason?: string;
    class_name: string;
    subject_name: string;
    original_teacher_name: string;
    substitute_teacher_name?: string;
    time_slot_name: string;
    start_time: string;
    end_time: string;
    created_at: string;
    updated_at: string;
}

interface BackendListResponse {
    items: BackendSubstitution[];
    total: number;
    page: number;
    limit: number;
}

function transformSubstitution(item: BackendSubstitution): TimetableSubstitution {
    return {
        id: item.id,
        schoolId: item.school_id,
        classTimetableSlotId: item.class_timetable_slot_id,
        substitutionDate: item.substitution_date,
        originalTeacherMembershipId: item.original_teacher_membership_id,
        substituteTeacherMembershipId: item.substitute_teacher_membership_id,
        status: item.status as TimetableSubstitution["status"],
        reason: item.reason,
        className: item.class_name,
        subjectName: item.subject_name,
        originalTeacherName: item.original_teacher_name,
        substituteTeacherName: item.substitute_teacher_name,
        timeSlotName: item.time_slot_name,
        startTime: item.start_time,
        endTime: item.end_time,
        createdAt: item.created_at,
        updatedAt: item.updated_at,
    };
}

export async function listSubstitutions(
    params: ListSubstitutionsParams = {}
): Promise<ListSubstitutionsResponse> {
    const searchParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === "") return;
        searchParams.set(key, String(value));
    });
    const response = await api.get<BackendListResponse>(
        `/api/timetable/substitutions?${searchParams.toString()}`
    );
    return {
        items: response.items.map(transformSubstitution),
        total: response.total,
        page: response.page,
        limit: response.limit,
    };
}

export async function getSubstitution(id: string): Promise<TimetableSubstitution> {
    const response = await api.get<{ substitution: BackendSubstitution }>(
        `/api/timetable/substitutions/${id}`
    );
    return transformSubstitution(response.substitution);
}

export async function createSubstitution(data: CreateSubstitutionRequest): Promise<{ id: string }> {
    return api.post<{ id: string }>("/api/timetable/substitutions", data);
}

export async function updateSubstitution(
    id: string,
    data: Omit<UpdateSubstitutionRequest, "id">
): Promise<void> {
    await api.patch(`/api/timetable/substitutions/${id}`, data);
}

export async function deleteSubstitution(id: string): Promise<void> {
    await api.delete(`/api/timetable/substitutions/${id}`);
}
