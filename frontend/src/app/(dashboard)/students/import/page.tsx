/**
 * /students/import — full-page student bulk import.
 *
 * Renders the StudentImportContainer component.
 */

"use client";

import { StudentImportContainer } from "@/features/student-import";

export default function StudentImportPage() {
    return (
        <div className="mx-auto max-w-5xl px-6 py-6">
            <StudentImportContainer />
        </div>
    );
}
