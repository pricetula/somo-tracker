"use client";

import { useSetupSummary } from "./hooks/use-setup-summary";
import { ProgressTracker } from "./progress-tracker";
import { AcademicPeriodColumn } from "./academic-period-column";
import { StreamsClassroomsColumn } from "./streams-classrooms-column";
import { EnrollmentsColumn } from "./enrollments-column";
import { Alert, AlertDescription } from "@/components/ui/alert";

export function SetupFlow() {
    const { data, isLoading, isError } = useSetupSummary();

    if (isLoading) return <div className="space-y-4">Loading...</div>;
    if (isError || !data)
        return (
            <Alert>
                <AlertDescription>Failed to load setup status</AlertDescription>
            </Alert>
        );

    const timelineLocked = data.academicPeriod.exists;
    const streamsExist = !!data.streams && data.streams.count > 0;
    const classroomsBuilt = !!data.classrooms && data.classrooms.count > 0;

    const steps = [
        { id: "timeline", label: "Timeline Locked", done: timelineLocked },
        { id: "streams", label: "Streams Defined", done: streamsExist },
        { id: "classrooms", label: "Classrooms Built", done: classroomsBuilt },
        {
            id: "enrollments",
            label: "Students Placed",
            done: !!data.enrollments?.items.some((i) => i.studentCount > 0),
        },
    ];

    return (
        <section className="space-y-6">
            <header className="flex justify-between">
                <h2>Classes & Streams</h2>
                <ProgressTracker steps={steps} />
            </header>
            <div className="grid gap-6 md:grid-cols-3">
                <AcademicPeriodColumn
                    exists={timelineLocked}
                    yearName={data.academicPeriod.yearName}
                    termName={data.academicPeriod.termName}
                    status={data.academicPeriod.status}
                />
                <StreamsClassroomsColumn
                    locked={!timelineLocked}
                    streamsCount={data.streams?.count}
                    items={data.classrooms?.items}
                    count={data.classrooms?.count}
                />
                <EnrollmentsColumn locked={!classroomsBuilt} items={data.enrollments?.items} />
            </div>
        </section>
    );
}
