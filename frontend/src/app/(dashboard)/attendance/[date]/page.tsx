/**
 * Intercepted route — Attendance marking sheet.
 *
 * Slides out from the right when the user clicks "Mark Attendance"
 * on a timeline slot. On hard refresh, the full page at
 * /attendance/mark/[slot_id]/[date] takes over.
 */

"use client";

import { SessionList } from "@/features/attendance/components/session-list";

export default SessionList;
