"use client";

import { TimetableTemplateWizard } from "@/features/timetable";

export default function TimetableAddPage() {
    return (
        <div className="p-6">
            <h1 className="mb-6 text-2xl font-semibold">Create Timetable Template</h1>
            <TimetableTemplateWizard />
        </div>
    );
}
