import { api } from "@/lib/api/client";
import type {
    CreateTimetableTemplatePayload,
    TimetableTemplate,
} from "../types/timetable-template";

export async function createTimetableTemplate(
    payload: CreateTimetableTemplatePayload
): Promise<TimetableTemplate> {
    const data = await api.post<TimetableTemplate>(`/api/timetable/templates`, payload);
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
