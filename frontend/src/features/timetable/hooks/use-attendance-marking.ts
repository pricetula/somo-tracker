/**
 * Attendance marking hook — fetches slot info, session status, and student marks
 * separately so the modal can pre-fill correctly (no forced PRESENT default).
 */
"use client";

import { useQuery } from "@tanstack/react-query";
import { getAllocation } from "@/lib/api/timetable";
import { getSessionsForSlot, getRecordsBySlot } from "@/lib/api/attendance";

export const attendanceMarkingKeys = {
    slot: (id: string) => ["attendance-marking", "slot", id] as const,
    session: (allocationId: string, date: string) =>
        ["attendance-marking", "session", allocationId, date] as const,
    records: (allocationId: string, date: string) =>
        ["attendance-marking", "records", allocationId, date] as const,
};

export function useSlotDetails(allocationId: string) {
    return useQuery({
        queryKey: attendanceMarkingKeys.slot(allocationId),
        queryFn: () => getAllocation(allocationId),
        enabled: !!allocationId,
    });
}

export function useSlotSession(allocationId: string, date: string) {
    return useQuery({
        queryKey: attendanceMarkingKeys.session(allocationId, date),
        queryFn: () => getSessionsForSlot(allocationId, date),
        enabled: !!allocationId && !!date,
    });
}

export function useSlotRecords(allocationId: string, date: string) {
    return useQuery({
        queryKey: attendanceMarkingKeys.records(allocationId, date),
        queryFn: () => getRecordsBySlot(allocationId, date),
        enabled: !!allocationId && !!date,
    });
}
