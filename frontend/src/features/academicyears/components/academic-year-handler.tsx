"use client";

import React from "react";
import { parseISO } from "date-fns";
import { Progress } from "@/components/ui/progress";
import { useAcademicYears } from "@/features/academicyears/hooks";
import type { AcademicYear, AcademicTerm } from "@/features/academicyears/types";
import { formatDate } from "@/lib/format-date";

const currentDate = new Date();

export function AcademicYearHandler() {
    const { data } = useAcademicYears();

    const academicYear: AcademicYear | null = React.useMemo(
        () => (data && data?.length ? data.find((yr) => yr.is_current) || null : null),
        [data]
    );

    const academicTerm: AcademicTerm | null = React.useMemo(() => {
        if (!academicYear || !academicYear.terms?.length) return null;

        const term = academicYear.terms.find((yr) => yr.is_current) || null;

        if (term) return term;

        return academicYear.terms.sort((a, b) => {
            const bb = parseISO(b.start_date).getTime();
            const aa = parseISO(a.start_date).getTime();
            return bb - aa;
        })[0];
    }, [academicYear]);

    const selectedAcademicTermPercentageProgress = React.useMemo(() => {
        const currentTime = currentDate.getTime();
        const startTime = academicTerm?.start_date
            ? parseISO(academicTerm.start_date).getTime()
            : 0;
        const endTime = academicTerm?.end_date ? parseISO(academicTerm.end_date).getTime() : 0;
        const progress = ((currentTime - startTime) / (endTime - startTime)) * 100;
        if (progress < 0 || !Number.isFinite(progress)) return 0;
        if (progress > 100) return 100;
        return progress;
    }, [academicTerm]);

    return (
        <section className="w-75">
            <div className="text-muted-foreground mb-2 text-xs">{academicYear?.name}</div>
            <div className="mb-4">{academicTerm?.name}</div>

            <div className="mb-4 flex justify-between text-xs">
                <span>{formatDate(academicTerm?.start_date || "")}</span>
                <span>{formatDate(academicTerm?.end_date || "")}</span>
            </div>

            <Progress
                value={selectedAcademicTermPercentageProgress}
                className="[&>div]:bg-green-500"
            />
        </section>
    );
}
