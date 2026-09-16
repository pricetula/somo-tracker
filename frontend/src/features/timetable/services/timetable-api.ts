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

export async function getTimetableTemplate(id: string): Promise<TimetableTemplate> {
    const data = await api.get<TimetableTemplate>(`/api/timetable/templates/${id}`);
    return data;
}
