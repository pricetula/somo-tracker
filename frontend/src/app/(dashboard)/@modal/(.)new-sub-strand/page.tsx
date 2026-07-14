/**
 * Intercepted route for creating a new sub-strand via the modal slot.
 */

import { Suspense } from "react";
import { ModalNewSubStrandContent } from "./content";

export default function NewSubStrandPage() {
    return (
        <Suspense fallback={null}>
            <ModalNewSubStrandContent />
        </Suspense>
    );
}
