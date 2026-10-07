import { useQuery } from "@tanstack/react-query";
import { listTeachers } from "@/lib/api/teachers";

export const teachersKeys = {
    list: (params?: { page?: number; limit?: number }) => ["teachers", params] as const,
};

export function useTeachers(params: { page?: number; limit?: number } = { page: 1, limit: 100 }) {
    return useQuery({
        queryKey: teachersKeys.list(params),
        queryFn: () => listTeachers(params),
        staleTime: 5 * 60 * 1000,
    });
}
