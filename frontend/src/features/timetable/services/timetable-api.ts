import { api } from "@/lib/api/client";
import type {
    ClassTimetableSlotWithDetails,
    CreateTimetableTemplatePayload,
    TimetableTemplate,
} from "../types/timetable-template";

export async function createTimetableTemplate(
    payload: CreateTimetableTemplatePayload
): Promise<{ id: string }> {
    const data = await api.post<{ id: string }>(`/api/timetable/templates`, payload);
    return data;
}

export async function getTimetableTemplates(): Promise<TimetableTemplate[]> {
    const data = await api.get<TimetableTemplate[]>(`/api/timetable/templates`);
    return data;
}

export async function getTimetableTemplate(id: string): Promise<TimetableTemplate> {
    const data = await api.get<TimetableTemplate>(`/api/timetable/templates/${id}`);
    return data;
}

export type TimeSlotResponse = {
    id: string;
    timetable_template_id: string;
    name: string;
    start_time: string;
    end_time: string;
    sequence_index: number;
    is_instructional: boolean;
};

export async function getTimeSlotsByTemplate(id: string): Promise<TimeSlotResponse[]> {
    const data = await api.get<TimeSlotResponse[]>(`/api/timetable/templates/${id}/slots`);
    return data;
}

export async function getClassTimetableSlots(
    templateId: string,
    classId: string
): Promise<ClassTimetableSlotWithDetails[]> {
    const data = await api.get<ClassTimetableSlotWithDetails[]>(
        `/api/timetable/templates/${templateId}/classes/${classId}/slots`
    );
    return data;
}

export async function deleteClassTimetableSlot(slotId: string): Promise<void> {
    await api.delete(`/api/timetable/class-timetable-slots/${slotId}`);
}
