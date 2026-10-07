import { useQuery } from "@tanstack/react-query";
import { listClasses } from "../services/api";

export const classesKeys = {
    list: (params?: { page?: number; limit?: number }) => ["classes", params] as const,
};

export function useClasses(params: { page?: number; limit?: number } = { page: 1, limit: 100 }) {
    return useQuery({
        queryKey: classesKeys.list(params),
        queryFn: () => listClasses(params),
        staleTime: 5 * 60 * 1000,
    });
}
