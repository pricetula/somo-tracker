/**
 * Dashboard summary hooks — lightweight queries that fetch counts,
 * pending items, setup progress, and recent activity for the dashboard.
 *
 * Each hook is focused on a single concern. They're composed in the
 * dashboard component, not in a monolithic "everything" hook.
 */

"use client";

import { useQuery } from "@tanstack/react-query";

import { listStudents } from "@/lib/api/students";
import { listTeachers } from "@/lib/api/teachers";
import { listClasses } from "@/lib/api/classes";
import { listAcademicYears } from "@/lib/api/academic-terms";
import { getInvitationCount } from "@/lib/api/invitations";
import { getActiveImportJob, listJobs } from "@/lib/api/imports";
import { listLearningAreas } from "@/lib/api/curriculum";
import { listTimeBlocks } from "@/lib/api/timetable-structure";
import { STALE_TIMES } from "@/lib/query-config";

// ─── Query keys ───────────────────────────────────────────────────────────

export const dashboardKeys = {
    counts: ["dashboard", "counts"] as const,
    pending: ["dashboard", "pending"] as const,
    setup: ["dashboard", "setup"] as const,
    activity: ["dashboard", "activity"] as const,
};

// ===========================================================================
// Hook: useDashboardCounts
// Fetches total counts of students, teachers, classes, and pending invitations.
// Uses limit=1 to read only the `total` field without fetching full records.
// ===========================================================================

export interface DashboardCounts {
    students: number;
    teachers: number;
    classes: number;
    pendingInvitations: number;
}

export function useDashboardCounts() {
    return useQuery<DashboardCounts>({
        queryKey: dashboardKeys.counts,
        queryFn: async () => {
            const [studentsRes, teachersRes, classesRes, invitationsRes] = await Promise.all([
                listStudents({ limit: 1 }),
                listTeachers({ limit: 1 }),
                listClasses({ limit: 1 }),
                getInvitationCount("TEACHER"),
            ]);

            return {
                students: studentsRes.total,
                teachers: teachersRes.total,
                classes: classesRes.total,
                pendingInvitations: invitationsRes.total,
            };
        },
        staleTime: STALE_TIMES.STANDARD,
    });
}

// ===========================================================================
// Hook: useDashboardPendingItems
// Fetches items needing attention: active imports, pending invitations detail.
// ===========================================================================

export interface PendingItem {
    type: "invitation" | "import" | "overcapacity" | "urgent_incident";
    label: string;
    description: string;
    count: number;
    href: string;
}

export function useDashboardPendingItems() {
    return useQuery<PendingItem[]>({
        queryKey: dashboardKeys.pending,
        queryFn: async () => {
            const results: PendingItem[] = [];

            // Active import job
            const activeImport = await getActiveImportJob();
            if (activeImport.active && activeImport.job) {
                results.push({
                    type: "import",
                    label: "Import in progress",
                    description: `${activeImport.job.job_type === "STUDENT_IMPORT" ? "Student" : "Staff"} import — ${activeImport.job.processed_records}/${activeImport.job.total_records} processed`,
                    count: activeImport.job.total_records - activeImport.job.processed_records,
                    href: `/imports/${activeImport.job.id}`,
                });
            }

            return results;
        },
        staleTime: STALE_TIMES.FREQUENT,
    });
}

// ===========================================================================
// Hook: useDashboardSetupProgress
// Checks which foundational items have been configured for the school.
// Returns an array of checklist items with completion status.
// ===========================================================================

export interface SetupChecklistItem {
    id: string;
    label: string;
    description: string;
    done: boolean;
    href: string;
}

export function useDashboardSetupProgress() {
    return useQuery<SetupChecklistItem[]>({
        queryKey: dashboardKeys.setup,
        queryFn: async () => {
            const [yearsRes, classesRes, studentsRes, areasRes, blocksRes] = await Promise.all([
                listAcademicYears(),
                listClasses({ limit: 1 }),
                listStudents({ limit: 1 }),
                listLearningAreas(),
                listTimeBlocks().catch(() => ({ items: [], total: 0 })),
            ]);

            const hasYears = (yearsRes.items?.length ?? 0) > 0;
            const hasClasses = classesRes.total > 0;
            const hasStudents = studentsRes.total > 0;
            const hasCurriculum = (areasRes.items?.length ?? 0) > 0;
            const hasTimetable = blocksRes.total > 0;

            return [
                {
                    id: "academic-year",
                    label: "Academic year",
                    description: "Set up the academic year and terms",
                    done: hasYears,
                    href: "/academic-years",
                },
                {
                    id: "classes",
                    label: "Classes",
                    description: "Create grade-level classes with streams",
                    done: hasClasses,
                    href: "/classes",
                },
                {
                    id: "students",
                    label: "Students",
                    description: "Enrol students into classes",
                    done: hasStudents,
                    href: "/students",
                },
                {
                    id: "curriculum",
                    label: "Curriculum",
                    description: "Set up learning areas, strands, and indicators",
                    done: hasCurriculum,
                    href: "/curriculum",
                },
                {
                    id: "timetable",
                    label: "Timetable",
                    description: "Configure timetable slots and allocate subjects",
                    done: hasTimetable,
                    href: "/timetable",
                },
            ];
        },
        staleTime: STALE_TIMES.STANDARD,
    });
}

// ===========================================================================
// Hook: useDashboardRecentActivity
// Fetches recent import/action history for the activity feed.
// ===========================================================================

export interface ActivityItem {
    id: string;
    type: "student_import" | "staff_invite" | "enrollment";
    label: string;
    description: string;
    timestamp: string;
    href?: string;
}

export function useDashboardRecentActivity() {
    return useQuery<ActivityItem[]>({
        queryKey: dashboardKeys.activity,
        queryFn: async () => {
            const jobsRes = await listJobs({ limit: 5 });

            const items: ActivityItem[] = jobsRes.items.map((job) => {
                const type = job.job_type === "STUDENT_IMPORT" ? "student_import" : "staff_invite";
                const succeeded = job.success_count > 0;
                const failed = job.failed_count > 0;
                let description = `${job.total_records} records`;

                if (succeeded && failed) {
                    description = `${job.success_count} succeeded, ${job.failed_count} failed`;
                } else if (succeeded) {
                    description = `${job.success_count} processed successfully`;
                } else if (failed) {
                    description = `${job.failed_count} failed`;
                }

                const statusLabel =
                    job.status === "completed"
                        ? "Completed"
                        : job.status === "completed_with_errors"
                          ? "Completed with errors"
                          : job.status === "failed"
                            ? "Failed"
                            : job.status === "processing"
                              ? "Processing"
                              : "Pending";

                const label =
                    type === "student_import"
                        ? `${statusLabel}: Student import`
                        : `${statusLabel}: Staff invitation`;

                return {
                    id: job.id,
                    type,
                    label,
                    description,
                    timestamp: job.created_at,
                    href: `/imports/${job.id}`,
                };
            });

            return items;
        },
        staleTime: STALE_TIMES.STANDARD,
    });
}
