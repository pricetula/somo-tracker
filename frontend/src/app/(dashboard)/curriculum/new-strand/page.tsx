/**
 * New Strand page.
 *
 * Full-page fallback when navigating directly to /curriculum/new-strand
 * or after a hard refresh (bypasses the intercepted modal route).
 */

import { Suspense } from "react";
import { NewStrandPageContent } from "./content";

export default function NewStrandPage() {
    return (
        <Suspense
            fallback={<div className="text-muted-foreground py-8 text-center">Loading...</div>}
        >
            <NewStrandPageContent />
        </Suspense>
    );
}
