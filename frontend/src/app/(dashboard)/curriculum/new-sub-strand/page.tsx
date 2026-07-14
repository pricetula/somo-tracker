/**
 * New Sub-Strand page.
 */

import { Suspense } from "react";
import { NewSubStrandPageContent } from "./content";

export default function NewSubStrandPage() {
    return (
        <Suspense
            fallback={<div className="text-muted-foreground py-8 text-center">Loading...</div>}
        >
            <NewSubStrandPageContent />
        </Suspense>
    );
}
