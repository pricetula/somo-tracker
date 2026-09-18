import { useQuery } from "@tanstack/react-query";
import { getClassTimetableSlots } from "../services/timetable-api";
import type { ClassTimetableSlotWithDetails } from "../types/timetable-template";

export const classTimetableSlotKeys = {
    byTemplateAndClass: (templateId: string, classId: string) =>
        ["timetable-class-slots", templateId, classId] as const,
};

export function useClassTimetableSlots(templateId: string, classId: string) {
    return useQuery<ClassTimetableSlotWithDetails[], Error>({
        queryKey: classTimetableSlotKeys.byTemplateAndClass(templateId, classId),
        queryFn: () => getClassTimetableSlots(templateId, classId),
        enabled: !!templateId && !!classId,
    });
}
