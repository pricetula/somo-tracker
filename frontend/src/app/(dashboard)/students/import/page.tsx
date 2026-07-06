/**
 * Students bulk import page — standalone route.
 *
 * Bulk import UI will be implemented later.
 */

import { StudentsImportForm } from "@/features/students/components/students-import/students-import";

export default function StudentsImportPage() {
    return <StudentsImportForm isDialogVersion={false} />;
}
