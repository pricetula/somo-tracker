"use client";

import {
  useQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import {
  fetchCurrentCalendar,
  saveAcademicCalendar,
} from "@/lib/api/academic-calendar";
import { getApiErrorMessage } from "@/lib/api/auth";
import type {
  AcademicYear,
  CreateAcademicCalendarPayload,
} from "@/features/calendar/types";

// ─── Query Keys ───────────────────────────────────────────────────────────

export const calendarKeys = {
  current: ["academic-calendar", "current"] as const,
};

// ─── Hooks ────────────────────────────────────────────────────────────────

/**
 * Fetch the current academic calendar.
 *
 * Cache settings per spec:
 *   - staleTime: Infinity (data changes exceptionally rarely)
 *   - refetchOnWindowFocus: false
 *   - Request deduplication is built into TanStack Query
 */
export function useAcademicCalendar() {
  return useQuery<AcademicYear | null>({
    queryKey: calendarKeys.current,
    queryFn: fetchCurrentCalendar,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    retry: 1,
  });
}

/**
 * Save (create or update) the academic calendar.
 *
 * On success:
 *   1. Show a checkmark / toast
 *   2. Invalidate the calendar query to re-run the decision tree
 */
export function useSaveAcademicCalendar() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateAcademicCalendarPayload) =>
      saveAcademicCalendar(payload),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: calendarKeys.current });
      toast.success("Academic calendar saved!", {
        description: "Your school calendar is now active.",
      });
    },
    onError: (err: unknown) => {
      toast.error("Failed to save calendar", {
        description: getApiErrorMessage(err),
      });
    },
  });
}
