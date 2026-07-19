/**
 * Student Detail Page — Full page render for /students/:id.
 *
 * Shows student profile, enrollment history, behavior notes, and
 * attendance summary for the current or selected term.
 */

"use client";

import { use, useState } from "react";
import Link from "next/link";
import {
    User,
    BookOpen,
    AlertTriangle,
    CalendarCheck,
    Loader2,
    ArrowUpRight,
    HeartPulse,
    BarChart3,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { StaticTable } from "@/components/shared/static-table";
import { useStudentDetail } from "@/features/students";
import { useStudentHistory } from "@/features/attendance";
import { useStudentHealth } from "@/features/health";

// Default start date for attendance history: 30 days before module load.
// Defined at module scope (not during render) to avoid impure function calls
// in the component body.
const THIRTY_DAYS_AGO = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split("T")[0];

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
    const { data: detailResponse, isLoading: detailLoading } = useStudentDetail(id);
    const detail = detailResponse?.data;

    // Fetch recent attendance (last 30 days by default)
    const { data: attendanceData } = useStudentHistory(id, {
        start_date: THIRTY_DAYS_AGO,
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
                    <TabsTrigger value="health">Health</TabsTrigger>
                    <TabsTrigger value="reports">Reports</TabsTrigger>
                    <TabsTrigger value="enrollments">Enrollments</TabsTrigger>
                </TabsList>

                {/* ── Overview Tab ─────────────────────────────────────── */}
                <TabsContent value="overview" className="space-y-6 pt-4">
                    <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
                        {/* Attendance summary card */}
                        <div className="bg-muted/30 p-4">
                            <div className="text-muted-foreground flex items-center gap-2 font-medium">
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
                </TabsContent>

                {/* ── Behavior Tab ─────────────────────────────────────── */}
                <TabsContent value="behavior" className="space-y-4 pt-4">
                    <div className="flex items-center justify-between">
                        <p className="text-muted-foreground">
                            {behaviorNotes.length > 0
                                ? `${behaviorNotes.length} note${behaviorNotes.length !== 1 ? "s" : ""} on record`
                                : "No behavior notes logged yet."}
                        </p>
                        {currentEnrollment && (
                            <Button variant="outline" size="sm" asChild>
                                <Link href={`/attendance/students/${id}`}>
                                    <CalendarCheck className="mr-1 h-3 w-3" />
                                    View Attendance
                                </Link>
                            </Button>
                        )}
                    </div>

                    {behaviorNotes.length === 0 ? (
                        <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                            <AlertTriangle className="h-8 w-8" />
                            <p className="font-medium">No behavior notes</p>
                            <p className="">
                                Teachers can log behavior notes during attendance marking. Approved
                                notes will appear here.
                            </p>
                        </div>
                    ) : (
                        behaviorNotes.map((note) => (
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
                        ))
                    )}
                </TabsContent>

                {/* ── Attendance Tab ───────────────────────────────────── */}
                <TabsContent value="attendance" className="space-y-4 pt-4">
                    {attendanceRecords.length === 0 ? (
                        <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
                            <CalendarCheck className="h-8 w-8" />
                            <p className="font-medium">No attendance records</p>
                            <p className="">Attendance history will appear here once marked.</p>
                        </div>
                    ) : (
                        <StaticTable
                            columns={[
                                { id: "date", header: "Date", cell: (rec) => rec.date },
                                {
                                    id: "status",
                                    header: "Status",
                                    cell: (rec) => (
                                        <Badge className={statusBadge[rec.status]?.className ?? ""}>
                                            {statusBadge[rec.status]?.label ?? rec.status}
                                        </Badge>
                                    ),
                                },
                                {
                                    id: "note",
                                    header: "Note",
                                    cell: (rec) => (
                                        <span className="text-muted-foreground text-xs">
                                            {rec.note ?? "—"}
                                        </span>
                                    ),
                                },
                            ]}
                            data={attendanceRecords.slice(0, 50)}
                            getRowId={(rec) => rec.id}
                            height={280}
                        />
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
                </TabsContent>
                {/* ── Health Tab ──────────────────────────────────────── */}
                <TabsContent value="health" className="space-y-4 pt-4">
                    <HealthTabContent studentId={id} />
                </TabsContent>

                {/* ── Reports Tab ─────────────────────────────────────── */}
                <TabsContent value="reports" className="space-y-4 pt-4">
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
                </TabsContent>
            </Tabs>
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
