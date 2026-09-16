import { useMutation, useQueryClient } from "@tanstack/react-query";
import { setupTimetableSlot } from "../services/timetable-assign-api";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const assignSlotKeys = {
    list: ["timetable-assign"] as const,
};

export function useAssignSlot() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationKey: [...assignSlotKeys.list, "create"],
        mutationFn: setupTimetableSlot,
        onSuccess: () => {
            toast.success("Timetable slot assigned");
            queryClient.invalidateQueries({ queryKey: ["timetable-time-slots"] });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
