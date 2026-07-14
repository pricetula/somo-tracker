/**
 * Student Detail Page — Full page render for /students/:id.
 *
 * Shows student profile, enrollment history, behavior notes, and
 * attendance summary for the current or selected term.
 */

"use client";

import { use, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { User, BookOpen, AlertTriangle, CalendarCheck, Loader2, ArrowUpRight } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { getStudent, type StudentDetail } from "@/lib/api/students";
import { getStudentHistory } from "@/lib/api/attendance";

interface Props {
    params: Promise<{ id: string }>;
}

const statusBadge: Record<string, { label: string; className: string }> = {
    PRESENT: {
        label: "Present",
        className: "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400",
    },
    ABSENT: {
        label: "Absent",
        className: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    },
    LATE: {
        label: "Late",
        className: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    },
    EXCUSED: {
        label: "Excused",
        className: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
    },
};

export default function StudentDetailPage({ params }: Props) {
    const { id } = use(params);
    const [activeTab, setActiveTab] = useState("overview");

    // Fetch student detail
    const { data: detail, isLoading: detailLoading } = useQuery<StudentDetail>({
        queryKey: ["student", id],
        queryFn: () => getStudent(id),
        enabled: !!id,
    });

    // Fetch recent attendance
    const { data: attendanceData } = useQuery({
        queryKey: ["attendance", "history", id],
        queryFn: () => getStudentHistory(id, {}),
        enabled: !!id,
    });

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
    const attendanceRecords = attendanceData?.items ?? [];

    return (
        <div className="space-y-8">
            {/* ── Header ─────────────────────────────────────────────── */}
            <div className="flex items-start justify-between">
                <div className="space-y-1">
                    <h1 className="text-foreground text-2xl font-bold">{student.full_name}</h1>
                    <div className="text-muted-foreground flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
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
                        <p className="text-muted-foreground text-sm">
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
                    <Button variant="outline" size="sm" asChild>
                        <Link href={`/attendance/students/${id}`}>
                            <CalendarCheck className="mr-1 h-4 w-4" />
                            Attendance
                        </Link>
                    </Button>
                </div>
            </div>

            {/* ── Tabs ───────────────────────────────────────────────── */}
            <Tabs value={activeTab} onValueChange={setActiveTab}>
                <TabsList>
                    <TabsTrigger value="overview">Overview</TabsTrigger>
                    <TabsTrigger value="behavior">
                        Behavior
                        {behaviorNotes.length > 0 && (
                            <Badge variant="secondary" className="ml-2">
                                {behaviorNotes.length}
                            </Badge>
                        )}
                    </TabsTrigger>
                    <TabsTrigger value="attendance">
                        Attendance
                        {attendanceRecords.length > 0 && (
                            <Badge variant="secondary" className="ml-2">
                                {attendanceRecords.length}
                            </Badge>
                        )}
                    </TabsTrigger>
                    <TabsTrigger value="enrollments">Enrollments</TabsTrigger>
                </TabsList>

                {/* ── Overview Tab ─────────────────────────────────────── */}
                <TabsContent value="overview" className="space-y-6 pt-4">
                    <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
                        {/* Attendance summary card */}
                        <div className="bg-muted/30 rounded-lg p-4">
                            <div className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
                                <CalendarCheck className="h-4 w-4" />
                                Attendance
                            </div>
                            <p className="text-foreground mt-2 text-2xl font-bold">
                                {attendanceRecords.length > 0 ? (
                                    <>
                                        {Math.round(
                                            (attendanceRecords.filter((r) => r.status === "PRESENT")
                                                .length /
                                                attendanceRecords.length) *
                                                100
                                        )}
                                        <span className="text-muted-foreground text-base font-normal">
                                            %
                                        </span>
                                    </>
                                ) : (
                                    <span className="text-muted-foreground text-base font-normal">
                                        —
                                    </span>
                                )}
                            </p>
                            <p className="text-muted-foreground text-xs">Recent periods</p>
                        </div>

                        {/* Behavior summary card */}
                        <div className="bg-muted/30 rounded-lg p-4">
                            <div className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
                                <AlertTriangle className="h-4 w-4" />
                                Behavior Notes
                            </div>
                            <p className="text-foreground mt-2 text-2xl font-bold">
                                {behaviorNotes.length > 0 ? (
                                    <>
                                        {behaviorNotes.filter((n) => n.is_urgent).length > 0 && (
                                            <span className="text-destructive">
                                                {behaviorNotes.filter((n) => n.is_urgent).length}{" "}
                                                urgent
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

                        {/* Enrollment summary card */}
                        <div className="bg-muted/30 rounded-lg p-4">
                            <div className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
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
                </TabsContent>

                {/* ── Behavior Tab ─────────────────────────────────────── */}
                <TabsContent value="behavior" className="space-y-4 pt-4">
                    {behaviorNotes.length === 0 ? (
                        <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                            <AlertTriangle className="h-8 w-8" />
                            <p className="font-medium">No behavior notes</p>
                            <p className="text-sm">Approved behavior notes will appear here.</p>
                        </div>
                    ) : (
                        behaviorNotes.map((note) => (
                            <div
                                key={note.id}
                                className={`rounded-lg border p-4 ${note.is_urgent ? "border-l-4 border-l-red-500" : "border-l-4 border-l-transparent"}`}
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
                                        <p className="text-foreground text-sm">
                                            {note.description}
                                        </p>
                                        <p className="text-muted-foreground text-xs">{note.date}</p>
                                    </div>
                                </div>
                            </div>
                        ))
                    )}
                </TabsContent>

                {/* ── Attendance Tab ───────────────────────────────────── */}
                <TabsContent value="attendance" className="space-y-4 pt-4">
                    {attendanceRecords.length === 0 ? (
                        <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                            <CalendarCheck className="h-8 w-8" />
                            <p className="font-medium">No attendance records</p>
                            <p className="text-sm">
                                Attendance history will appear here once marked.
                            </p>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-muted-foreground border-b text-left">
                                        <th className="pb-2 font-medium">Date</th>
                                        <th className="pb-2 font-medium">Status</th>
                                        <th className="pb-2 font-medium">Note</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {attendanceRecords.slice(0, 50).map((rec) => (
                                        <tr key={rec.id} className="border-b last:border-b-0">
                                            <td className="text-foreground py-2">{rec.date}</td>
                                            <td className="py-2">
                                                <Badge
                                                    className={
                                                        statusBadge[rec.status]?.className ?? ""
                                                    }
                                                >
                                                    {statusBadge[rec.status]?.label ?? rec.status}
                                                </Badge>
                                            </td>
                                            <td className="text-muted-foreground py-2 text-xs">
                                                {rec.note ?? "—"}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                    <Button variant="outline" size="sm" asChild>
                        <Link href={`/attendance/students/${id}`}>
                            <ArrowUpRight className="mr-1 h-4 w-4" />
                            Full attendance history
                        </Link>
                    </Button>
                </TabsContent>

                {/* ── Enrollments Tab ──────────────────────────────────── */}
                <TabsContent value="enrollments" className="space-y-4 pt-4">
                    {enrollments.length === 0 ? (
                        <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                            <BookOpen className="h-8 w-8" />
                            <p className="font-medium">No enrollments</p>
                            <p className="text-sm">Enrollment history will appear here.</p>
                        </div>
                    ) : (
                        <div className="overflow-x-auto">
                            <table className="w-full text-sm">
                                <thead>
                                    <tr className="text-muted-foreground border-b text-left">
                                        <th className="pb-2 font-medium">Term</th>
                                        <th className="pb-2 font-medium">Class</th>
                                        <th className="pb-2 font-medium">Status</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {enrollments.map((enr) => (
                                        <tr key={enr.id} className="border-b last:border-b-0">
                                            <td className="text-foreground py-2">
                                                {enr.term_name} {enr.academic_year}
                                            </td>
                                            <td className="text-foreground py-2">
                                                {enr.class_name}
                                            </td>
                                            <td className="py-2">
                                                <Badge variant="secondary">{enr.status}</Badge>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    )}
                </TabsContent>
            </Tabs>
        </div>
    );
}
