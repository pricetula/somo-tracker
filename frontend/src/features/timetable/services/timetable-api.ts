import { api } from "@/lib/api/client";
import type {
    CreateTimetableTemplatePayload,
    TimetableTemplate,
} from "../types/timetable-template";

export async function createTimetableTemplate(
    schoolId: string,
    payload: CreateTimetableTemplatePayload
): Promise<TimetableTemplate> {
    const data = await api.post<TimetableTemplate>(
        `/api/v1/schools/${schoolId}/timetable-templates`,
        payload
    );
    return data;
}

export async function getTimetableTemplate(
    schoolId: string,
    id: string
): Promise<TimetableTemplate> {
    const data = await api.get<TimetableTemplate>(
        `/api/v1/schools/${schoolId}/timetable-templates/${id}`
    );
    return data;
}
