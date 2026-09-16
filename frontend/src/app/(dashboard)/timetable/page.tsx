"use client";

import { TimetableTemplateWizard } from "@/features/timetable";

export default function TimetablePage() {
    return (
        <div className="p-6">
            <h1 className="mb-6 text-2xl font-semibold">Timetable Templates</h1>
            <TimetableTemplateWizard />
        </div>
    );
}
