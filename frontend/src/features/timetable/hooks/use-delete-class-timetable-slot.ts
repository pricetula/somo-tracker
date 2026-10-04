import { useMutation, useQueryClient } from "@tanstack/react-query";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";
import { deleteClassTimetableSlot } from "../services/timetable-api";
import { classTimetableSlotKeys } from "./use-class-timetable-slots";

export function useDeleteClassTimetableSlot(templateId: string, classId: string) {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: ["timetable", "delete-slot", templateId, classId],
        mutationFn: (slotId: string) => deleteClassTimetableSlot(slotId),
        onSuccess: () => {
            queryClient.invalidateQueries({
                queryKey: classTimetableSlotKeys.byTemplateAndClass(templateId, classId),
            });
            toast.success("Slot deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
