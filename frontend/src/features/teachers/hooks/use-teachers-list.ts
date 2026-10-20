import { useQuery } from "@tanstack/react-query";
import { listTeachers } from "@/lib/api/teachers";

export const teachersKeys = {
    list: (params?: { page?: number; limit?: number }) => ["teachers", params] as const,
};

export function teachersListQueryFn() {
    return (params: {
        page?: number;
        limit?: number;
        search?: string;
        filters?: Record<string, string | string[]>;
        invitation_status?: string;
    }) => {
        const invitationStatus =
            typeof params.filters?.status === "string"
                ? params.filters.status
                : params.invitation_status;
        return listTeachers({
            page: params.page,
            limit: params.limit,
            search: params.search,
            invitation_status:
                invitationStatus === "all" || !invitationStatus
                    ? undefined
                    : (invitationStatus as "invited" | "accepted"),
        });
    };
}

export function useTeachers(params: { page?: number; limit?: number } = { page: 1, limit: 100 }) {
    return useQuery({
        queryKey: teachersKeys.list(params),
        queryFn: () => listTeachers(params),
        staleTime: 5 * 60 * 1000,
    });
}
