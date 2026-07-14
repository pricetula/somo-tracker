/**
 * Intercepted route — Student detail rendered as a sliding side sheet.
 *
 * When a user clicks a student name in a table, this sheet slides out
 * from the right keeping the list visible but dimmed.
 * On hard refresh the full page at /students/[id] takes over.
 */

"use client";

import { use } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Loader2, User, AlertTriangle, CalendarCheck, BookOpen } from "lucide-react";

import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";

import { getStudent, type StudentDetail } from "@/lib/api/students";

interface Props {
    params: Promise<{ id: string }>;
}

export default function StudentDetailSheet({ params }: Props) {
    const router = useRouter();
    const { id } = use(params);

    const { data: detail, isLoading } = useQuery<StudentDetail>({
        queryKey: ["student", id],
        queryFn: () => getStudent(id),
        enabled: !!id,
    });

    return (
        <Sheet
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <SheetContent
                side="right"
                className="w-full overflow-y-auto data-[side=right]:sm:max-w-xl"
            >
                <SheetHeader>
                    <SheetTitle>Student Profile</SheetTitle>
                </SheetHeader>

                {isLoading ? (
                    <div className="flex items-center justify-center py-16">
                        <Loader2 className="text-muted-foreground h-5 w-5 animate-spin" />
                    </div>
                ) : !detail ? (
                    <div className="flex flex-col items-center gap-2 py-16">
                        <User className="text-muted-foreground h-10 w-10" />
                        <p className="text-muted-foreground font-medium">Student not found</p>
                    </div>
                ) : (
                    <div className="space-y-6 pt-6">
                        {/* Header */}
                        <div className="space-y-1">
                            <h2 className="text-foreground text-xl font-bold">
                                {detail.full_name}
                            </h2>
                            <div className="text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-xs">
                                {detail.admission_number && (
                                    <span>Adm: {detail.admission_number}</span>
                                )}
                                {detail.upi_number && <span>UPI: {detail.upi_number}</span>}
                                {detail.gender && (
                                    <span>Gender: {detail.gender === "M" ? "Male" : "Female"}</span>
                                )}
                            </div>
                            {detail.class_name && (
                                <p className="text-muted-foreground text-xs">{detail.class_name}</p>
                            )}
                        </div>

                        {/* Quick stats */}
                        <div className="grid grid-cols-3 gap-3">
                            <div className="bg-muted/30 rounded-lg p-3 text-center">
                                <CalendarCheck className="text-muted-foreground mx-auto mb-1 h-4 w-4" />
                                <p className="text-muted-foreground text-xs">Enrollments</p>
                                <p className="text-foreground font-bold">
                                    {detail.enrollments?.length ?? 0}
                                </p>
                            </div>
                            <div className="bg-muted/30 rounded-lg p-3 text-center">
                                <AlertTriangle className="text-muted-foreground mx-auto mb-1 h-4 w-4" />
                                <p className="text-muted-foreground text-xs">Behavior</p>
                                <p className="text-foreground font-bold">
                                    {detail.behavior?.length ?? 0}
                                </p>
                            </div>
                            <div className="bg-muted/30 rounded-lg p-3 text-center">
                                <BookOpen className="text-muted-foreground mx-auto mb-1 h-4 w-4" />
                                <p className="text-muted-foreground text-xs">Active</p>
                                <Badge
                                    variant="secondary"
                                    className={
                                        detail.is_active
                                            ? "bg-emerald-100 text-xs text-emerald-700"
                                            : ""
                                    }
                                >
                                    {detail.is_active ? "Yes" : "No"}
                                </Badge>
                            </div>
                        </div>

                        {/* Behavior notes */}
                        {detail.behavior && detail.behavior.length > 0 && (
                            <div className="space-y-2">
                                <h3 className="text-foreground text-sm font-medium">
                                    Behavior Notes
                                </h3>
                                {detail.behavior.slice(0, 5).map((note) => (
                                    <div
                                        key={note.id}
                                        className={`rounded-lg border p-3 ${note.is_urgent ? "border-l-4 border-l-red-500" : ""}`}
                                    >
                                        <div className="flex items-center gap-2">
                                            <Badge variant="outline" className="text-xs">
                                                {note.category_name}
                                            </Badge>
                                            {note.is_urgent && (
                                                <Badge variant="destructive" className="text-xs">
                                                    Urgent
                                                </Badge>
                                            )}
                                        </div>
                                        <p className="text-foreground mt-1 text-sm">
                                            {note.description}
                                        </p>
                                        <p className="text-muted-foreground mt-1 text-xs">
                                            {note.date}
                                        </p>
                                    </div>
                                ))}
                            </div>
                        )}

                        {/* Recent enrollments */}
                        {detail.enrollments && detail.enrollments.length > 0 && (
                            <div className="space-y-2">
                                <h3 className="text-foreground text-sm font-medium">Enrollments</h3>
                                <div className="space-y-1">
                                    {detail.enrollments.slice(0, 5).map((e) => (
                                        <div
                                            key={e.id}
                                            className="text-muted-foreground flex items-center justify-between text-xs"
                                        >
                                            <span>
                                                {e.term_name} {e.academic_year}
                                            </span>
                                            <span>{e.class_name}</span>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}

                        {/* View full profile link */}
                        <a
                            href={`/students/${id}`}
                            className="text-primary text-sm font-medium hover:underline"
                            onClick={(e) => {
                                e.preventDefault();
                                router.push(`/students/${id}`);
                            }}
                        >
                            View full profile →
                        </a>
                    </div>
                )}
            </SheetContent>
        </Sheet>
    );
}
