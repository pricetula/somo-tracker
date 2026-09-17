import { useQuery } from "@tanstack/react-query";
import { getTimetableTemplate } from "../services/timetable-api";
import type { TimetableTemplate } from "../types/timetable-template";

export const timetableTemplateDetailKeys = {
    byId: (id: string) => ["timetable-templates", "detail", id] as const,
};

export function useTimetableTemplate(id: string) {
    return useQuery<TimetableTemplate, Error>({
        queryKey: timetableTemplateDetailKeys.byId(id),
        queryFn: () => getTimetableTemplate(id),
        enabled: !!id,
    });
}
