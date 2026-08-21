"use client";

import { useQueryClient } from "@tanstack/react-query";
import {
    User,
    BookOpen,
    AlertTriangle,
    Loader2,
    ArrowUpRight,
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
// import { useUnlinkStudent } from "@/features/parents";
import Link from "next/link";

interface StudentDetailContentProps {
    studentId: string;
    variant?: "page" | "sheet";
    onDeleteSuccess: () => void;
}

import { HealthSection } from "./health-section";

export function StudentDetailContent({
    studentId,
    variant = "page",
    onDeleteSuccess,
}: StudentDetailContentProps) {
    const isCompact = variant === "sheet";

    const { data: detailResponse, isLoading: detailLoading } = useStudentDetail(studentId);
    const queryClient = useQueryClient();
    const deleteMutation = useDeleteStudent();
    // const unlinkStudentMutation = useUnlinkStudent();
    const detail = detailResponse?.data;

    const emDash = "\u2014";

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

    const enrollments = detail.enrollments ?? [];
    const currentEnrollment = enrollments[0];
    const behaviorNotes = detail.behavior ?? [];
    const linkedParents = detail.linked_parents ?? [];

    const handleUnlinkParent = async (parentId: string, parentName: string) => {
        if (!window.confirm(`Unlink ${parentName} from this student?`)) return;
        try {
            // await unlinkStudentMutation.mutateAsync({ parentId, studentId });
            queryClient.invalidateQueries({ queryKey: ["students", "detail", studentId] });
        } catch {
            // handled by hook onError
        }
    };

    const Heading = isCompact ? "h2" : "h1";
    const outerGap = isCompact ? "space-y-6" : "space-y-8";
    const tableParentHeight = isCompact ? 200 : 280;
    const tableEnrollHeight = isCompact ? 160 : 200;
    const emptyPadding = isCompact ? "py-8" : "py-12";
    const overviewCols = isCompact ? "grid-cols-2" : "grid-cols-1 md:grid-cols-2";

    return (
        <div className={outerGap}>
            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="flex items-start justify-between">
                <div className="space-y-1">
                    <Heading className="text-foreground text-2xl font-bold">
                        {detail.full_name}
                    </Heading>
                    <div className="text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
                        {detail.admission_number && (
                            <span>
                                Adm: <span className="font-mono">{detail.admission_number}</span>
                            </span>
                        )}
                        {detail.upi_number && (
                            <span>
                                UPI: <span className="font-mono">{detail.upi_number}</span>
                            </span>
                        )}
                        {detail.knec_assessment_number && (
                            <span>
                                KNEC:{" "}
                                <span className="font-mono">{detail.knec_assessment_number}</span>
                            </span>
                        )}
                        <span>Gender: {detail.gender === "M" ? "Male" : "Female"}</span>
                    </div>
                    {currentEnrollment && (
                        <p className="text-muted-foreground text-xs">
                            {currentEnrollment.class_name} &middot; {currentEnrollment.term_name}{" "}
                            {currentEnrollment.academic_year}
                            <Badge variant="secondary" className="ml-2 text-xs">
                                {currentEnrollment.status}
                            </Badge>
                        </p>
                    )}
                </div>
                <div className="flex shrink-0 items-center gap-2">
                    <AlertDialog>
                        <AlertDialogTrigger>
                            <Button variant="outline" size="sm" className="text-destructive">
                                <Trash2 className="size-3.5" />
                                {isCompact ? null : "Delete"}
                            </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                            <AlertDialogHeader>
                                <AlertDialogTitle>Delete Student</AlertDialogTitle>
                                <AlertDialogDescription>
                                    Are you sure you want to delete &ldquo;{detail.full_name}
                                    &rdquo;? This action cannot be undone.
                                </AlertDialogDescription>
                            </AlertDialogHeader>
                            <AlertDialogFooter>
                                <AlertDialogCancel>Cancel</AlertDialogCancel>
                                <AlertDialogAction
                                    variant="destructive"
                                    onClick={async () => {
                                        try {
                                            await deleteMutation.mutateAsync(studentId);
                                            onDeleteSuccess();
                                        } catch {
                                            // handled by hook onError
                                        }
                                    }}
                                    disabled={deleteMutation.isPending}
                                >
                                    {deleteMutation.isPending ? "Deleting\u2026" : "Delete"}
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
                <div className={`grid ${overviewCols} gap-6`}>
                    {/* Behavior summary */}
                    <div className="bg-muted/30 p-4">
                        <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium">
                            <AlertTriangle className="h-4 w-4" />
                            Behavior Notes
                        </div>
                        <p className="text-foreground mt-2 text-2xl font-bold">
                            {behaviorNotes.length > 0 ? (
                                <>
                                    {behaviorNotes.filter((n) => n.is_urgent).length > 0 ? (
                                        <span className="text-destructive">
                                            {behaviorNotes.filter((n) => n.is_urgent).length} urgent
                                        </span>
                                    ) : (
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
                        <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium">
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
                            Latest: {currentEnrollment?.class_name ?? emDash}
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
                    <div
                        className={`text-muted-foreground flex flex-col items-center gap-2 ${emptyPadding}`}
                    >
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
                <HealthSection studentId={studentId} isCompact={isCompact} />
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
                    <Button variant="outline" size="sm">
                        <Link href={`/students/${studentId}/link-parent`}>
                            <Link2 className="mr-1.5 size-3.5" />
                            Link Parent
                        </Link>
                    </Button>
                </div>

                {linkedParents.length === 0 ? (
                    <div
                        className={`text-muted-foreground flex flex-col items-center gap-2 ${emptyPadding}`}
                    >
                        <Users className="h-8 w-8" />
                        <p className="font-medium">No linked parents</p>
                        <Button variant="outline" size="sm" className="mt-2">
                            <Link href={`/students/${studentId}/link-parent`}>
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
                            ...(!isCompact
                                ? [
                                      {
                                          id: "email" as const,
                                          header: "Email" as const,
                                          cell: (lp: (typeof linkedParents)[number]) => (
                                              <span className="text-muted-foreground">
                                                  {lp.email}
                                              </span>
                                          ),
                                      },
                                      {
                                          id: "phone" as const,
                                          header: "Phone" as const,
                                          cell: (lp: (typeof linkedParents)[number]) => (
                                              <span className="text-muted-foreground">
                                                  {lp.phone_number}
                                              </span>
                                          ),
                                      },
                                      {
                                          id: "relationship" as const,
                                          header: "Relationship" as const,
                                          cell: (lp: (typeof linkedParents)[number]) => (
                                              <span className="text-muted-foreground">
                                                  {lp.relationship || emDash}
                                              </span>
                                          ),
                                      },
                                  ]
                                : [
                                      {
                                          id: "phone" as const,
                                          header: "Phone" as const,
                                          cell: (lp: (typeof linkedParents)[number]) => (
                                              <span className="text-muted-foreground">
                                                  {lp.phone_number}
                                              </span>
                                          ),
                                      },
                                  ]),
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
                                        <span className="text-muted-foreground">{emDash}</span>
                                    ),
                            },
                            {
                                id: "actions",
                                header: "",
                                width: isCompact ? "48px" : "64px",
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
                        height={tableParentHeight}
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
                    <div
                        className={`text-muted-foreground flex flex-col items-center gap-2 ${emptyPadding}`}
                    >
                        <BookOpen className="h-8 w-8" />
                        <p className="font-medium">No enrollments</p>
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
                        height={tableEnrollHeight}
                    />
                )}
            </section>

            {/* ── Reports ─────────────────────────────────────────────── */}
            <section>
                <h2 className="text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase">
                    Reports
                </h2>
                <div
                    className={`text-muted-foreground flex flex-col items-center gap-2 ${emptyPadding}`}
                >
                    <BarChart3 className="h-8 w-8" />
                    <p className="font-medium">Student Reports</p>
                    <p className="text-xs">Generate and view term reports for this student.</p>
                    <Button variant="outline" size="sm" className="mt-2">
                        <Link href={`/reports/student/${studentId}`}>
                            <ArrowUpRight className="mr-1 h-4 w-4" />
                            View Reports
                        </Link>
                    </Button>
                </div>
            </section>
        </div>
    );
}
