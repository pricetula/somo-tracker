"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import { eventApi } from "../services/event-api";
import type { CreateEventInput } from "../types/event.types";

export function useUpdateEvent() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: ({ id, input }: { id: string; input: CreateEventInput }) =>
            eventApi.update(id, input),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["events"] });
            toast.success("Event updated");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
