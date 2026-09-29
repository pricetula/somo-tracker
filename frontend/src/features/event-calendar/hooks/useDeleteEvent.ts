"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import { eventApi } from "../services/event-api";

export function useDeleteEvent() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => eventApi.delete(id),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["events"] });
            toast.success("Event deleted");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
