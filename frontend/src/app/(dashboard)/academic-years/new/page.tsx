/**
 * Create Academic Year page.
 */

import { AcademicYearForm } from "@/features/academic-years";

export default function NewAcademicYearPage() {
    return (
        <div className="max-w-lg space-y-6">
            <div>
                <h1 className="text-foreground text-2xl font-semibold">Create Academic Year</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Define a new academic year with a name and date range.
                </p>
            </div>
            <AcademicYearForm />
        </div>
    );
}
