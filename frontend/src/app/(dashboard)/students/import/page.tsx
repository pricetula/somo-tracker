/**
 * Students bulk import page — standalone route.
 *
 * Wraps StudentsImportForm in a consistent layout matching the teacher/staff
 * import page pattern (max-w-2xl, back button).
 */

"use client";

import { StudentsImportForm } from "@/features/students/components/students-import/students-import";

export default function StudentsImportPage() {
    return (
        <div className="mx-auto flex max-w-2xl flex-col gap-6 px-6 pt-6 pb-8">
            <StudentsImportForm isDialogVersion={false} />
        </div>
    );
}
