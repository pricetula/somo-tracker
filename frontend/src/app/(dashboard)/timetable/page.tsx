"use client";

import { TimetableTemplateWizard } from "@/features/timetable";

// TODO: Replace with real schoolId from auth/session context
const SCHOOL_ID = "demo-school-id";

export default function TimetablePage() {
    return (
        <div className="p-6">
            <h1 className="mb-6 text-2xl font-semibold">Timetable Templates</h1>
            <TimetableTemplateWizard schoolId={SCHOOL_ID} />
        </div>
    );
}
