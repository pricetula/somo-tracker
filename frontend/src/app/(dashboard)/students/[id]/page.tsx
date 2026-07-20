/**
 * Student Detail Page — Full page render for /students/:id.
 *
 * Shows student profile, enrollment history, behavior notes,
 * health information, and report links.
 */

"use client";

import { use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import {
    User,
    BookOpen,
    AlertTriangle,
    Loader2,
    ArrowUpRight,
    HeartPulse,
    BarChart3,
    Trash2,
    Link2,
    UserPlus,
    Users,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

import { StaticTable } from "@/components/shared/static-table";
import { useStudentDetail, useDeleteStudent } from "@/features/students";
import { useStudentHealth } from "@/features/health";
import { useUnlinkStudent } from "@/features/parents";

interface Props {
    params: Promise<{ id: string }>;
}

export default function StudentDetailPage({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    // Fetch student detail
    const { data: detailResponse, isLoading: detailLoading } = useStudentDetail(id);
    const queryClient = useQueryClient();
    const deleteMutation = useDeleteStudent();
    const unlinkStudentMutation = useUnlinkStudent();
    const detail = detailResponse?.data;

    if (detailLoading) {
        return (
            <div className="flex items-center justify-center py-24">
                <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
            </div>
        );
    }

    if (!detail) {
        return (
            <div className="flex flex-col items-center gap-2 py-24">
                <User className="text-muted-foreground h-12 w-12" />
                <p className="text-muted-foreground font-medium">Student not found</p>
            </div>
        );
    }

    const student = detail;
    const enrollments = student.enrollments ?? [];
    const currentEnrollment = enrollments[0]; // most recent
    const behaviorNotes = student.behavior ?? [];
    const linkedParents = student.linked_parents ?? [];

    const handleUnlinkParent = async (parentId: string, parentName: string) => {
        if (!window.confirm(`Unlink ${parentName} from this student?`)) {
            return;
        }
        try {
            await unlinkStudentMutation.mutateAsync({ parentId, studentId: id });
            queryClient.invalidateQueries({ queryKey: ["students", "detail", id] });
        } catch {
            // handled by hook onError
        }
    };

    return (
        <div className="space-y-8">
            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="flex items-start justify-between">
                <div className="space-y-1">
                    <h1 className="text-foreground text-2xl font-bold">{student.full_name}</h1>
                    <div className="text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1">
                        {student.admission_number && (
                            <span>
                                Adm: <span className="font-mono">{student.admission_number}</span>
                            </span>
                        )}
                        {student.upi_number && (
                            <span>
                                UPI: <span className="font-mono">{student.upi_number}</span>
                            </span>
                        )}
                        {student.knec_assessment_number && (
                            <span>
                                KNEC:{" "}
                                <span className="font-mono">{student.knec_assessment_number}</span>
                            </span>
                        )}
                        <span>Gender: {student.gender === "M" ? "Male" : "Female"}</span>
                    </div>
                    {currentEnrollment && (
                        <p className="text-muted-foreground">
                            {currentEnrollment.class_name} &middot; {currentEnrollment.term_name}{" "}
                            {currentEnrollment.academic_year}
                            <Badge variant="secondary" className="ml-2 text-xs">
                                {currentEnrollment.status}
                            </Badge>
                        </p>
                    )}
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" asChild>
                        <Link href={`/students/${id}/edit`}>Edit profile</Link>
                    </Button>
                    <AlertDialog>
                        <AlertDialogTrigger asChild>
                            <Button variant="outline" size="sm" className="text-destructive">
                                <Trash2 className="size-3.5" />
                                Delete
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Delete Student</AlertDialogTitle>
                                <AlertDialogDescription>
                                    Are you sure you want to delete &ldquo;{student.full_name}
                                    &rdquo;? This action cannot be undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                    variant="destructive"
                                    onClick={async () => {
                                        try {
                                            await deleteMutation.mutateAsync(id);
                                            router.push("/students");
                                        } catch {
                                            // handled by hook onError
                                        }
                                    }}
                                    disabled={deleteMutation.isPending}
                                >
                                    {deleteMutation.isPending ? "Deleting…" : "Delete"}
                                </AlertDialogAction>
                            </AlertDialogFooter>
                        </AlertDialogContent>
                    </AlertDialog>
                </div>
            </div>

            {/* ── Overview ─────────────────────────────────────────── */}
            <section>
                <h2 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase">
                    Overview
                </h2>
                <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
                    {/* Behavior summary */}
                    <div className="bg-muted/30 p-4">
                        <div className="text-muted-foreground flex items-center gap-2 font-medium">
                            <AlertTriangle className="h-4 w-4" />
                            Behavior Notes
                        </div>
                        <p className="text-foreground mt-2 text-2xl font-bold">
                            {behaviorNotes.length > 0 ? (
                                <>
                                    {behaviorNotes.filter((n) => n.is_urgent).length > 0 && (
                                        <span className="text-destructive">
                                            {behaviorNotes.filter((n) => n.is_urgent).length} urgent
                                        </span>
                                    )}
                                    {behaviorNotes.filter((n) => n.is_urgent).length === 0 && (
                                        <>{behaviorNotes.length} notes</>
                                    )}
                                </>
                            ) : (
                                <span className="text-muted-foreground text-base font-normal">
                                    No notes
                                </span>
                            )}
                        </p>
                    </div>

                    {/* Enrollment summary */}
                    <div className="bg-muted/30 p-4">
                        <div className="text-muted-foreground flex items-center gap-2 font-medium">
                            <BookOpen className="h-4 w-4" />
                            Enrollments
                        </div>
                        <p className="text-foreground mt-2 text-2xl font-bold">
                            {enrollments.length}{" "}
                            <span className="text-muted-foreground text-base font-normal">
                                terms
                            </span>
                        </p>
                        <p className="text-muted-foreground text-xs">
                            Latest: {currentEnrollment?.class_name ?? "—"}
                        </p>
                    </div>
                </div>
            </section>

            {/* ── Behavior Notes ─────────────────────────────────────── */}
            <section>
                <h2 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase">
                    Behavior Notes
                    {behaviorNotes.length > 0 && (
                        <span className="ml-2 font-normal">({behaviorNotes.length})</span>
                    )}
                </h2>

                {behaviorNotes.length === 0 ? (
                    <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                        <AlertTriangle className="h-8 w-8" />
                        <p className="font-medium">No behavior notes</p>
                    </div>
                ) : (
                    <div className="space-y-2">
                        {behaviorNotes.map((note) => (
                            <div
                                key={note.id}
                                className={`bg-muted/30 p-4 ${note.is_urgent ? "border-l-destructive border-l-2" : ""}`}
                            >
                                <div className="flex items-start justify-between">
                                    <div className="space-y-1">
                                        <div className="flex items-center gap-2">
                                            <Badge variant="outline">{note.category_name}</Badge>
                                            {note.is_urgent && (
                                                <Badge variant="destructive" className="gap-1">
                                                    <AlertTriangle className="h-3 w-3" />
                                                    Urgent
                                                </Badge>
                                            )}
                                            <Badge variant="secondary" className="text-xs">
                                                {note.status}
                                            </Badge>
                                        </div>
                                        <p className="text-foreground">{note.description}</p>
                                        <p className="text-muted-foreground text-xs">{note.date}</p>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </section>

            {/* ── Health ──────────────────────────────────────────────── */}
            <section>
                <h2 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase">
                    Health
                </h2>
                <HealthTabContent studentId={id} />
            </section>

            {/* ── Parents ────────────────────────────────────────────── */}
            <section>
                <div className="mb-3 flex items-center justify-between">
                    <h2 className="text-muted-foreground text-xs font-semibold tracking-wider uppercase">
                        Parents
                        {linkedParents.length > 0 && (
                            <span className="ml-2 font-normal">({linkedParents.length})</span>
                        )}
                    </h2>
                    <Button variant="outline" size="sm" asChild>
                        <Link href={`/students/${id}/link-parent`}>
                            <Link2 className="mr-1.5 size-3.5" />
                            Link Parent
                        </Link>
                    </Button>
                </div>

                {linkedParents.length === 0 ? (
                    <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                        <Users className="h-8 w-8" />
                        <p className="font-medium">No linked parents</p>
                        <p className="">
                            Link a parent or guardian to manage family relationships.
                        </p>
                        <Button variant="outline" size="sm" className="mt-2" asChild>
                            <Link href={`/students/${id}/link-parent`}>
                                <UserPlus className="mr-1.5 size-3.5" />
                                Link Parent
                            </Link>
                        </Button>
                    </div>
                ) : (
                    <StaticTable
                        columns={[
                            {
                                id: "name",
                                header: "Parent Name",
                                cell: (lp) => <span className="font-medium">{lp.full_name}</span>,
                            },
                            {
                                id: "email",
                                header: "Email",
                                cell: (lp) => (
                                    <span className="text-muted-foreground">{lp.email}</span>
                                ),
                            },
                            {
                                id: "phone",
                                header: "Phone",
                                cell: (lp) => (
                                    <span className="text-muted-foreground">{lp.phone_number}</span>
                                ),
                            },
                            {
                                id: "relationship",
                                header: "Relationship",
                                cell: (lp) => (
                                    <span className="text-muted-foreground">
                                        {lp.relationship || "—"}
                                    </span>
                                ),
                            },
                            {
                                id: "primary",
                                header: "Primary",
                                cell: (lp) =>
                                    lp.is_primary ? (
                                        <Badge
                                            variant="secondary"
                                            className="bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400"
                                        >
                                            Primary
                                        </Badge>
                                    ) : (
                                        <span className="text-muted-foreground">—</span>
                                    ),
                            },
                            {
                                id: "actions",
                                header: "",
                                width: "64px",
                                cell: (lp) => (
                                    <Button
                                        variant="ghost"
                                        size="icon-sm"
                                        onClick={() =>
                                            handleUnlinkParent(lp.parent_id, lp.full_name)
                                        }
                                        title="Unlink parent"
                                    >
                                        <Trash2 className="text-destructive size-3.5" />
                                        <span className="sr-only">Unlink</span>
                                    </Button>
                                ),
                            },
                        ]}
                        data={linkedParents}
                        getRowId={(lp) => lp.parent_id}
                        height={280}
                    />
                )}
            </section>

            {/* ── Enrollments ──────────────────────────────────────────── */}
            <section>
                <h2 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase">
                    Enrollments
                    {enrollments.length > 0 && (
                        <span className="ml-2 font-normal">({enrollments.length})</span>
                    )}
                </h2>

                {enrollments.length === 0 ? (
                    <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                        <BookOpen className="h-8 w-8" />
                        <p className="font-medium">No enrollments</p>
                        <p className="">Enrollment history will appear here.</p>
                    </div>
                ) : (
                    <StaticTable
                        columns={[
                            {
                                id: "term",
                                header: "Term",
                                cell: (enr) => `${enr.term_name} ${enr.academic_year}`,
                            },
                            { id: "class", header: "Class", cell: (enr) => enr.class_name },
                            {
                                id: "status",
                                header: "Status",
                                cell: (enr) => <Badge variant="secondary">{enr.status}</Badge>,
                            },
                        ]}
                        data={enrollments}
                        getRowId={(enr) => enr.id}
                        height={200}
                    />
                )}
            </section>

            {/* ── Reports ─────────────────────────────────────────────── */}
            <section>
                <h2 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase">
                    Reports
                </h2>
                <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                    <BarChart3 className="h-8 w-8" />
                    <p className="font-medium">Student Reports</p>
                    <p className="">Generate and view term reports for this student.</p>
                    <Button variant="outline" size="sm" asChild className="mt-2">
                        <Link href={`/reports/student/${id}`}>
                            <ArrowUpRight className="mr-1 h-4 w-4" />
                            View Reports
                        </Link>
                    </Button>
                </div>
            </section>
        </div>
    );
}

// ─── Health Tab Content ────────────────────────────────────────────────────

function HealthTabContent({ studentId }: { studentId: string }) {
    const { data: healthData, isLoading, isError } = useStudentHealth(studentId);

    if (isLoading) {
        return (
            <div className="flex items-center justify-center py-12">
                <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
            </div>
        );
    }

    if (isError) {
        return (
            <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                <HeartPulse className="h-8 w-8" />
                <p className="font-medium">Failed to load health data</p>
            </div>
        );
    }

    const incidents = healthData?.incidents ?? [];
    const profile = healthData?.profile;

    return (
        <div className="space-y-6">
            {/* Health Profile */}
            {profile && (
                <div className="bg-muted/30 p-4">
                    <h3 className="mb-2 font-semibold">Health Profile</h3>
                    <div className="space-y-1">
                        {profile.blood_group && (
                            <p>
                                <span className="text-muted-foreground">Blood Group:</span>{" "}
                                {profile.blood_group}
                            </p>
                        )}
                        {profile.allergies && profile.allergies.length > 0 && (
                            <p>
                                <span className="text-muted-foreground">Allergies:</span>{" "}
                                {profile.allergies.join(", ")}
                            </p>
                        )}
                        {profile.chronic_conditions && profile.chronic_conditions.length > 0 && (
                            <p>
                                <span className="text-muted-foreground">Chronic Conditions:</span>{" "}
                                {profile.chronic_conditions.join(", ")}
                            </p>
                        )}
                        {profile.emergency_instructions && (
                            <p>
                                <span className="text-muted-foreground">Emergency Notes:</span>{" "}
                                {profile.emergency_instructions}
                            </p>
                        )}
                    </div>
                </div>
            )}

            {/* Medical Incidents */}
            <div>
                <div className="mb-3 flex items-center justify-between">
                    <h3 className="font-semibold">
                        Medical Incidents
                        {incidents.length > 0 && (
                            <span className="text-muted-foreground ml-2 font-normal">
                                ({incidents.length})
                            </span>
                        )}
                    </h3>
                    <Button variant="outline" size="sm" asChild>
                        <Link href={`/health/students/${studentId}`}>
                            <ArrowUpRight className="mr-1 h-3 w-3" />
                            Full History
                        </Link>
                    </Button>
                </div>

                {incidents.length === 0 ? (
                    <div className="text-muted-foreground flex flex-col items-center gap-2 py-8">
                        <HeartPulse className="h-8 w-8" />
                        <p className="font-medium">No medical incidents</p>
                        <p className="">No health incidents have been logged for this student.</p>
                    </div>
                ) : (
                    <div className="space-y-2">
                        {incidents.slice(0, 10).map((incident) => (
                            <div key={incident.id} className="bg-muted/30 p-3">
                                <div className="flex items-start justify-between">
                                    <div className="space-y-1">
                                        <p className="font-medium">{incident.symptoms}</p>
                                        <p className="text-muted-foreground text-xs">
                                            {new Date(
                                                incident.incident_timestamp
                                            ).toLocaleDateString("en-US", {
                                                month: "short",
                                                day: "numeric",
                                                year: "numeric",
                                                hour: "2-digit",
                                                minute: "2-digit",
                                            })}
                                            {incident.logged_by_name &&
                                                ` \u00b7 ${incident.logged_by_name}`}
                                        </p>
                                        {incident.action_taken && (
                                            <p className="text-muted-foreground mt-1 text-xs">
                                                Action: {incident.action_taken}
                                            </p>
                                        )}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
}
