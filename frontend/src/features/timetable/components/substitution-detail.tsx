"use client";

import { useEffect, useState } from "react";
import { getSubstitution } from "../services/substitutions-api";
import type { TimetableSubstitution } from "../types/substitution";

interface SubstitutionDetailProps {
    id: string;
}

export function SubstitutionDetail({ id }: SubstitutionDetailProps) {
    const [sub, setSub] = useState<TimetableSubstitution | null>(null);

    useEffect(() => {
        getSubstitution(id).then(setSub);
    }, [id]);

    if (!sub) {
        return (
            <div className="space-y-6 p-6">
                <p className="text-muted-foreground">Loading…</p>
            </div>
        );
    }

    const statusConfig: Record<string, { label: string; className: string }> = {
        PENDING: { label: "Pending", className: "bg-amber-100 text-amber-800" },
        ASSIGNED: { label: "Assigned", className: "bg-blue-100 text-blue-800" },
        COMPLETED: { label: "Completed", className: "bg-emerald-100 text-emerald-800" },
        CANCELLED: { label: "Cancelled", className: "bg-rose-100 text-rose-800" },
    };

    const { label, className } = statusConfig[sub.status] || {
        label: sub.status,
        className: "bg-muted text-muted-foreground",
    };

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">
                    {sub.className} — {sub.subjectName}
                </h1>
                <p className="text-muted-foreground">ID: {sub.id}</p>
            </div>
            <div className="space-y-2">
                <div className="grid grid-cols-2 gap-4">
                    <div>
                        <span className="font-medium">Date:</span> {sub.substitutionDate}
                    </div>
                    <div>
                        <span className="font-medium">Time:</span> {sub.timeSlotName} (
                        {sub.startTime}–{sub.endTime})
                    </div>
                    <div>
                        <span className="font-medium">Original Teacher:</span>{" "}
                        {sub.originalTeacherName}
                    </div>
                    <div>
                        <span className="font-medium">Substitute:</span>{" "}
                        {sub.substituteTeacherName ?? "—"}
                    </div>
                    <div>
                        <span className="font-medium">Status:</span>
                        <span
                            className={`ml-2 inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${className}`}
                        >
                            {label}
                        </span>
                    </div>
                    {sub.reason && (
                        <div className="col-span-2">
                            <span className="font-medium">Reason:</span> {sub.reason}
                        </div>
                    )}
                    <div>
                        <span className="font-medium">Created:</span> {sub.createdAt}
                    </div>
                    <div>
                        <span className="font-medium">Updated:</span> {sub.updatedAt}
                    </div>
                </div>
            </div>
        </div>
    );
}
