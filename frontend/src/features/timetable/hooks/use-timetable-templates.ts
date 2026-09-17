import { api } from "@/lib/api/client";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { createTimetableTemplate, getTimetableTemplates } from "../services/timetable-api";
import type {
    CreateTimetableTemplatePayload,
    TimetableTemplate,
} from "../types/timetable-template";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export function useUpdateTimetableTemplate() {
    return useMutation<
        { message: string },
        Error,
        { id: string; name: string; description: string }
    >({
        mutationFn: ({ id, name, description }) =>
            api.patch(`/api/timetable/templates/${id}`, { name, description }) as Promise<{
                message: string;
            }>,
        onError(err) {
            toast.error(getErrorMessage(err));
        },
    });
}

export const timetableTemplateKeys = {
    list: ["timetable-templates", "list"] as const,
    create: ["timetable-templates", "create"] as const,
};

export function useTimetableTemplates() {
    return useQuery<TimetableTemplate[], Error>({
        queryKey: timetableTemplateKeys.list,
        queryFn: getTimetableTemplates,
    });
}

export function useCreateTimetableTemplate() {
    const queryClient = useQueryClient();

    return useMutation<{ id: string }, Error, CreateTimetableTemplatePayload>({
        mutationKey: timetableTemplateKeys.create,
        mutationFn: (payload) => createTimetableTemplate(payload),
        async onMutate(payload) {
            await queryClient.cancelQueries({ queryKey: timetableTemplateKeys.list });
            const previous = queryClient.getQueryData<TimetableTemplate[]>(
                timetableTemplateKeys.list
            );
            const optimistic: TimetableTemplate = {
                id: `optimistic-${Date.now()}`,
                name: payload.name,
                description: payload.description ?? "",
                school_id: "", // filled by server on success
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
            } as TimetableTemplate;
            if (previous) {
                queryClient.setQueryData<TimetableTemplate[]>(timetableTemplateKeys.list, [
                    optimistic,
                    ...previous,
                ]);
            }
            return { previous };
        },
        onError(err, _variables, context) {
            if (context?.previous) {
                queryClient.setQueryData(timetableTemplateKeys.list, context.previous);
            }
            toast.error(getErrorMessage(err));
        },
        onSettled() {
            queryClient.invalidateQueries({ queryKey: timetableTemplateKeys.list });
        },
        onSuccess: () => {
            toast.success("Timetable template saved");
        },
    });
}
