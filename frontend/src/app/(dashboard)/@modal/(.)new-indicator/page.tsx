/**
 * Intercepted route for creating a new performance indicator via the modal slot.
 */

import { Suspense } from "react";
import { ModalNewIndicatorContent } from "./content";

export default function NewIndicatorPage() {
    return (
        <Suspense fallback={null}>
            <ModalNewIndicatorContent />
        </Suspense>
    );
}
