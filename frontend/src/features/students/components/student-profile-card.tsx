/**
 * Student Profile Card — displays student demographics.
 *
 * Shows: Name, Gender, DOB, UPI, KNEC#, Status, Created date.
 */

"use client";

import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import type { StudentDetail } from "../types";

// ─── Props ─────────────────────────────────────────────────────────────────

interface StudentProfileCardProps {
    detail: StudentDetail | undefined;
    isLoading: boolean;
}

// ─── Helpers ───────────────────────────────────────────────────────────────

function formatDate(date: string | null | undefined): string {
    if (!date) return "—";
    return new Date(date).toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: "numeric",
    });
}

// ─── Field Row ────────────────────────────────────────────────────────────

function FieldRow({ label, value }: { label: string; value: string }) {
    return (
        <div className="flex items-baseline gap-4 py-1.5">
            <span className="text-muted-foreground w-32 shrink-0 text-xs font-medium">{label}</span>
            <span className="text-sm">{value}</span>
        </div>
    );
}

// ─── Component ─────────────────────────────────────────────────────────────

export function StudentProfileCard({ detail, isLoading }: StudentProfileCardProps) {
    if (isLoading || !detail) {
        return (
            <div className="bg-muted/30 space-y-3 rounded-md p-4">
                <Skeleton className="h-6 w-48" />
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-4 w-36" />
                <Skeleton className="h-4 w-44" />
            </div>
        );
    }

    return (
        <div className="bg-muted/30 rounded-md p-4">
            <h2 className="mb-3 text-lg font-semibold">{detail.full_name}</h2>
            <div className="space-y-0.5">
                <FieldRow
                    label="Gender"
                    value={
                        detail.gender === "M"
                            ? "Male"
                            : detail.gender === "F"
                              ? "Female"
                              : detail.gender || "—"
                    }
                />
                <FieldRow label="Date of Birth" value={formatDate(detail.date_of_birth)} />
                <FieldRow label="UPI Number" value={detail.upi_number || "—"} />
                <FieldRow label="KNEC Assessment #" value={detail.knec_assessment_number || "—"} />
                <FieldRow label="Enrolled" value={formatDate(detail.created_at)} />
                <div className="flex items-baseline gap-4 py-1.5">
                    <span className="text-muted-foreground w-32 shrink-0 text-xs font-medium">
                        Status
                    </span>
                    <Badge
                        variant="secondary"
                        className={
                            detail.is_active
                                ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                                : "bg-muted text-muted-foreground"
                        }
                    >
                        {detail.is_active ? "Active" : "Inactive"}
                    </Badge>
                </div>
            </div>
        </div>
    );
}
