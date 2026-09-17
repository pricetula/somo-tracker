import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteClassTimetableSlot } from "../services/timetable-api";
import { classTimetableSlotKeys } from "./use-class-timetable-slots";

export function useDeleteClassTimetableSlot(templateId: string, classId: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (slotId: string) => deleteClassTimetableSlot(slotId),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: classTimetableSlotKeys.byTemplateAndClass(templateId, classId),
            });
        },
    });
}
