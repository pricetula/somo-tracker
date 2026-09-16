import { useQuery } from "@tanstack/react-query";
import { getTimeSlotsByTemplate, type TimeSlotResponse } from "../services/timetable-api";

export const timeSlotKeys = {
    byTemplate: (id: string) => ["timetable-time-slots", id] as const,
};

export function useTimeSlots(templateId: string) {
    return useQuery<TimeSlotResponse[], Error>({
        queryKey: timeSlotKeys.byTemplate(templateId),
        queryFn: () => getTimeSlotsByTemplate(templateId),
        enabled: !!templateId,
    });
}
