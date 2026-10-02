import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
    listSubstitutions,
    getSubstitution,
    createSubstitution,
    updateSubstitution,
    deleteSubstitution,
} from "../services/substitutions-api";
import type {
    ListSubstitutionsParams,
    CreateSubstitutionRequest,
    UpdateSubstitutionRequest,
} from "../types/substitution";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const substitutionKeys = {
    list: (params: ListSubstitutionsParams) =>
        ["timetable", "substitutions", "list", params] as const,
    detail: (id: string) => ["timetable", "substitutions", "detail", id] as const,
};

export function useSubstitutions(params: ListSubstitutionsParams = {}) {
    return useQuery({
        queryKey: substitutionKeys.list(params),
        queryFn: () => listSubstitutions(params),
        staleTime: 30_000,
    });
}

export function useSubstitution(id: string) {
    return useQuery({
        queryKey: substitutionKeys.detail(id),
        queryFn: () => getSubstitution(id),
        enabled: !!id,
    });
}

export function useCreateSubstitution() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (data: CreateSubstitutionRequest) => createSubstitution(data),
        onSuccess: () => {
            toast.success("Substitution created");
            queryClient.invalidateQueries({ queryKey: ["timetable", "substitutions", "list"] });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useUpdateSubstitution() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: Omit<UpdateSubstitutionRequest, "id"> }) =>
            updateSubstitution(id, data),
        onSuccess: (_data, variables) => {
            toast.success("Substitution updated");
            queryClient.invalidateQueries({ queryKey: ["timetable", "substitutions", "list"] });
            queryClient.invalidateQueries({ queryKey: substitutionKeys.detail(variables.id) });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useDeleteSubstitution() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (id: string) => deleteSubstitution(id),
        onSuccess: () => {
            toast.success("Substitution deleted");
            queryClient.invalidateQueries({ queryKey: ["timetable", "substitutions", "list"] });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}
