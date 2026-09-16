import { useQuery } from "@tanstack/react-query";
import { listSubjects } from "../services/api";

export const subjectsKeys = {
    list: (params?: { page?: number; limit?: number }) => ["subjects", params] as const,
};

export function useSubjects(params: { page?: number; limit?: number } = { page: 1, limit: 100 }) {
    return useQuery({
        queryKey: subjectsKeys.list(params),
        queryFn: () => listSubjects(params),
        staleTime: 5 * 60 * 1000,
    });
}
