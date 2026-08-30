"use client";

import Link from "next/link";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useLowestAttendanceStudents } from "../hooks/use-lowest-attendance-students";

function getInitials(firstName: string, lastName: string) {
    return `${firstName?.[0]?.toUpperCase?.()}${lastName?.[0]?.toUpperCase?.()}`;
}

export function LowestAttendanceStudents() {
    const { data: studentList = [], isLoading, isError } = useLowestAttendanceStudents(5);

    if (isLoading) {
        return (
            <article className="p-4">
                <header className="mb-8">Students with lowest attendance</header>
                <ul className="space-y-4">
                    {[...Array(5)].map((_, i) => (
                        <li
                            key={i}
                            className="flex animate-pulse items-center gap-4 border-b border-dashed pb-4 last:border-b-0"
                        >
                            <Avatar>
                                <AvatarFallback className="bg-muted" />
                            </Avatar>
                            <div className="bg-muted h-4 w-32 rounded" />
                            <div className="bg-muted ml-auto h-4 w-20 rounded" />
                        </li>
                    ))}
                </ul>
            </article>
        );
    }

    if (isError) {
        return (
            <article className="p-4">
                <header className="mb-8">Students with lowest attendance</header>
                <p className="text-destructive py-4 text-center">Failed to load data</p>
            </article>
        );
    }

    if (!studentList?.length) {
        return (
            <article className="p-4">
                <header className="mb-8">Students with lowest attendance</header>
                <p className="text-muted-foreground py-8 text-center">
                    No attendance data for this week
                </p>
            </article>
        );
    }

    return (
        <article className="p-4">
            <header className="mb-8">Students with lowest attendance</header>
            <ul>
                {studentList.map((s) => (
                    <li
                        key={s.student_id}
                        className="mb-4 flex items-center gap-4 border-b border-dashed pb-4 last:border-b-0"
                    >
                        <Avatar>
                            <AvatarFallback>
                                {getInitials(s.first_name, s.last_name)}
                            </AvatarFallback>
                        </Avatar>
                        <Link href={`/students/${s.student_id}`}>
                            {s.first_name} {s.last_name}
                        </Link>
                        <span className="ml-auto flex flex-col items-end">
                            <span className="mb-2 text-rose-500">{s.attendance_percentage}%</span>
                            <span className="text-muted-foreground">
                                <b>
                                    {s.present_count}/{s.total_periods}
                                </b>{" "}
                                Periods
                            </span>
                        </span>
                    </li>
                ))}
            </ul>
        </article>
    );
}
