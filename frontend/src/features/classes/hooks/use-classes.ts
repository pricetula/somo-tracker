import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listClasses, getClass, createClass } from "../services/api";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

export const classesKeys = {
    list: (params?: { page?: number; limit?: number }) => ["classes", params] as const,
    detail: (id: string) => ["class", id] as const,
};

export function classesListQueryFn() {
    return (params: {
        page?: number;
        limit?: number;
        search?: string;
        filters?: Record<string, string | string[]>;
    }) => listClasses(params);
}

export function useClasses(params: { page?: number; limit?: number } = { page: 1, limit: 100 }) {
    return useQuery({
        queryKey: classesKeys.list(params),
        queryFn: () => listClasses(params),
        staleTime: 5 * 60 * 1000,
    });
}

export function useClass(id: string) {
    return useQuery({
        queryKey: classesKeys.detail(id),
        queryFn: () => getClass(id),
        enabled: !!id,
        staleTime: 5 * 60 * 1000,
    });
}

export function useCreateClass() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: ["classes", "create"],
        mutationFn: (data: { name: string; gradeId: string; streamId?: string }) =>
            createClass(data),
        onSuccess: () => {
            toast.success("Class created");
            queryClient.invalidateQueries({ queryKey: classesKeys.list() });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}

export function useCreateClassesBatch() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationKey: ["classes", "create-batch"],
        mutationFn: async (items: { name: string; gradeId: string; streamId?: string }[]) => {
            for (const item of items) {
                await createClass(item);
            }
        },
        onSuccess: () => {
            toast.success("Classes created successfully");
            queryClient.invalidateQueries({ queryKey: classesKeys.list() });
        },
        onError: (err) => toast.error(getErrorMessage(err)),
    });
}
