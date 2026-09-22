"use client";

import { useEffect, useState } from "react";
import { getClass } from "../services/api";
import type { ClassDetail } from "../types/class";

interface ClassDetailProps {
    id: string;
}

export function ClassDetail({ id }: ClassDetailProps) {
    const [detail, setDetail] = useState<ClassDetail | null>(null);

    useEffect(() => {
        getClass(id).then(setDetail);
    }, [id]);

    if (!detail) return null;

    return (
        <div className="space-y-6 p-6">
            <div className="space-y-1">
                <h1 className="text-2xl font-semibold">{detail.name}</h1>
                <p className="text-muted-foreground">
                    {detail.grade} • {detail.stream} • {detail.academicYear}
                </p>
            </div>
            <div className="space-y-2">
                <div>
                    <span className="font-medium">Teacher:</span> {detail.teacherName ?? "—"}
                </div>
                <div>
                    <span className="font-medium">Students:</span> {detail.studentsCount}
                </div>
                {detail.description && (
                    <div>
                        <span className="font-medium">Description:</span> {detail.description}
                    </div>
                )}
            </div>
        </div>
    );
}
