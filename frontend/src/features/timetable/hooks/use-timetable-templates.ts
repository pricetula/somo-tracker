import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createTimetableTemplate } from "../services/timetable-api";
import type {
    CreateTimetableTemplatePayload,
    TimetableTemplate,
} from "../types/timetable-template";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const timetableTemplateKeys = {
    list: ["timetable-templates", "list"] as const,
    create: ["timetable-templates", "create"] as const,
};

export function useCreateTimetableTemplate() {
    const queryClient = useQueryClient();

    return useMutation<TimetableTemplate, Error, CreateTimetableTemplatePayload>({
        mutationKey: timetableTemplateKeys.create,
        mutationFn: (payload) => createTimetableTemplate(payload),
        onSuccess: (data) => {
            toast.success("Timetable template saved");
            queryClient.invalidateQueries({ queryKey: timetableTemplateKeys.list });
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
