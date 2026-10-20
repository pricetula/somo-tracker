import { useQuery } from "@tanstack/react-query";
import { listAttendanceSessions } from "../services/api";
import type { ListAttendanceSessionsParams } from "../types/attendance";

export const attendanceKeys = {
    sessions: (params: ListAttendanceSessionsParams) => ["attendance", "sessions", params] as const,
};

export function attendanceListQueryFn() {
    return (params: ListAttendanceSessionsParams) => listAttendanceSessions(params);
}

export function useAttendanceSessions(params: ListAttendanceSessionsParams = {}) {
    return useQuery({
        queryKey: attendanceKeys.sessions(params),
        queryFn: () => listAttendanceSessions(params),
        staleTime: 30_000,
    });
}
