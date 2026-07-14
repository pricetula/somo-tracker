/**
 * Intercepted route for creating a new strand via the modal slot.
 */

import { Suspense } from "react";
import { ModalNewStrandContent } from "./content";

export default function NewStrandPage() {
    return (
        <Suspense fallback={null}>
            <ModalNewStrandContent />
        </Suspense>
    );
}
