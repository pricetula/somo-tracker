/**
 * New Performance Indicator page.
 */

import { Suspense } from "react";
import { NewIndicatorPageContent } from "./content";

export default function NewIndicatorPage() {
    return (
        <Suspense
            fallback={<div className="text-muted-foreground py-8 text-center">Loading...</div>}
        >
            <NewIndicatorPageContent />
        </Suspense>
    );
}
