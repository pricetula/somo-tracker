/**
 * Attendance status enum for the timetable marking UI.
 * Mirrors the backend AttendanceStatus values exactly.
 */
import React from "react";
import { Check, X, Clock, ShieldCheck } from "lucide-react";

export type AttendanceStatus = "PRESENT" | "ABSENT" | "LATE" | "EXCUSED";

export const ATTENDANCE_STATUSES: ReadonlyArray<{
    value: AttendanceStatus;
    label: string;
    activeClass: string;
    icon: React.ReactNode;
}> = [
    {
        value: "PRESENT",
        label: "Present",
        activeClass: "bg-emerald-100 text-emerald-700 border-emerald-200",
        icon: <Check size={12} aria-hidden />,
    },
    {
        value: "ABSENT",
        label: "Absent",
        activeClass: "bg-rose-100 text-rose-700 border-rose-200",
        icon: <X size={12} aria-hidden />,
    },
    {
        value: "LATE",
        label: "Late",
        activeClass: "bg-amber-100 text-amber-700 border-amber-200",
        icon: <Clock size={12} aria-hidden />,
    },
    {
        value: "EXCUSED",
        label: "Excused",
        activeClass: "bg-blue-100 text-blue-700 border-blue-200",
        icon: <ShieldCheck size={12} aria-hidden />,
    },
] as const;
