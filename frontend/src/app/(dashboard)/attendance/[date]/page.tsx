/**
 * Intercepted route — Attendance marking sheet.
 *
 * Slides out from the right when the user clicks "Mark Attendance"
 * on a timeline slot. On hard refresh, the full page at
 * /attendance/mark/[slot_id]/[date] takes over.
 */

"use client";

import { AttendanceTimeline } from "@/features/attendance";

export default AttendanceTimeline;
