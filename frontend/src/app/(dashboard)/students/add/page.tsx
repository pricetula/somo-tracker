/**
 * Intercepted route for student bulk import via the modal slot.
 *
 * Renders an empty dialog overlay — bulk import UI will be implemented later.
 */

"use client";

import { StudentsImportForm } from "@/features/students/components/students-import/students-import";

export default function StudentsImport() {
    return <StudentsImportForm isDialogVersion={false} />;
}
