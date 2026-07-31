"use client";

import { Button } from "@/components/ui/button";
import { type AcademicTerm } from "@/lib/api/academic-terms";

export function TermActionsCell({
    term,
    onEdit,
}: {
    term: AcademicTerm;
    onEdit: (term: AcademicTerm) => void;
}) {
    return (
        <div className="flex items-center justify-end">
            <Button variant="outline" size="sm" onClick={() => onEdit(term)}>
                Edit
            </Button>
        </div>
    );
}
