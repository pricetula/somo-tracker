"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import Link from "next/link";

type EnrollmentItem = {
    classId: string;
    className: string;
    grade: string;
    stream: string;
    studentCount: number;
};

type Props = {
    locked: boolean;
    items?: EnrollmentItem[];
};

export function EnrollmentsColumn({ locked, items }: Props) {
    if (locked) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Classroom Enrollments</CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground text-sm">Build classrooms first.</p>
                </CardContent>
            </Card>
        );
    }

    if (!items || items.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Classroom Enrollments</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                    <p className="text-muted-foreground text-sm">No students placed yet.</p>
                    <Link href="/classes" className={buttonVariants()}>
                        Assign Student Rosters
                    </Link>
                </CardContent>
            </Card>
        );
    }

    const empty = items.filter((i) => i.studentCount === 0);

    return (
        <Card>
            <CardHeader>
                <CardTitle>Classroom Enrollments</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
                {empty.length === 0 ? (
                    <>
                        <p className="text-muted-foreground text-sm">
                            All classes have students enrolled.
                        </p>
                        <Link
                            href="/classes"
                            className={buttonVariants({ variant: "outline", size: "sm" })}
                        >
                            View Enrollments
                        </Link>
                    </>
                ) : (
                    <div className="max-h-30 space-y-1 overflow-y-auto text-sm">
                        {empty.map((i) => (
                            <Link
                                key={i.classId}
                                href={`/classes/${i.classId}`}
                                className="block pb-2"
                            >
                                <span className="font-medium">{i.className}</span>
                            </Link>
                        ))}
                    </div>
                )}
            </CardContent>
        </Card>
    );
}
