"use client";

import Link from "next/link";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

function getInitials(firstName: string, lastName: string) {
    return `${firstName?.[0]?.toUpperCase?.()}${lastName?.[0]?.toUpperCase?.()}`;
}

export function LowestAttendanceStudents() {
    const data = {
        week_start_date: "2026-08-17",
        data: [
            {
                student_id: "a1b2c3d4-e5f6-7890-abcd-ef0123456789",
                first_name: "Brian",
                last_name: "Kiptoo",
                total_periods: 25,
                present_count: 15,
                attendance_percentage: 60.0,
            },
            {
                student_id: "b2c3d4e5-f6a1-8901-bcde-f0123456789a",
                first_name: "Mercy",
                last_name: "Wanjiku",
                total_periods: 25,
                present_count: 17,
                attendance_percentage: 68.0,
            },
            {
                student_id: "c3d4e5f6-a1b2-9012-cdef-0123456789ab",
                first_name: "Kevin",
                last_name: "Otieno",
                total_periods: 25,
                present_count: 18,
                attendance_percentage: 72.0,
            },
            {
                student_id: "d4e5f6a1-b2c3-0123-def0-123456789abc",
                first_name: "Amina",
                last_name: "Hassan",
                total_periods: 25,
                present_count: 19,
                attendance_percentage: 76.0,
            },
            {
                student_id: "e5f6a1b2-c3d4-1234-ef01-23456789abcd",
                first_name: "John",
                last_name: "Mwangi",
                total_periods: 25,
                present_count: 20,
                attendance_percentage: 80.0,
            },
        ],
    };

    const studentList = data?.data || [];

    return (
        <article className="p-4 text-xs">
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
                            <span className="mb-2 text-sm text-rose-500">
                                {s.attendance_percentage}%
                            </span>
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
