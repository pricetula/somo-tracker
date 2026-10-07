"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/errors";
import { eventApi } from "../services/event-api";
import type { CreateEventInput } from "../types/event.types";

export function useCreateEvent() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: (input: CreateEventInput) => eventApi.create(input),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["events"] });
            toast.success("Event created");
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });
}
