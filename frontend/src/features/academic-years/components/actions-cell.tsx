"use client";

import { Button } from "@/components/ui/button";
import { type AcademicYear } from "@/lib/api/academic-terms";
import Link from "next/link";

/**
 * Year row actions.
 *
 * Academic years are read-only via the API (year creation/activation is driven
 * by the term lifecycle) — so the only year-level action is opening the year
 * detail page, where terms can be created/edited/activated/deleted.
 */
export function ActionsCell({ year }: { year: AcademicYear }) {
    return (
        <div className="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" asChild>
                <Link href={`/academic-years/${year.id}`}>Edit</Link>
            </Button>
        </div>
    );
}
