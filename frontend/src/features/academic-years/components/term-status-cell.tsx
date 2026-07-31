"use client";

import { Badge } from "@/components/ui/badge";
import { type AcademicTerm } from "@/lib/api/academic-terms";

export function TermStatusCell({ term }: { term: AcademicTerm }) {
    return term.is_current ? (
        <Badge variant="default">Current</Badge>
    ) : (
        <span className="text-muted-foreground">Scheduled</span>
    );
}
