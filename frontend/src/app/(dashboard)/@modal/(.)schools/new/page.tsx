/**
 * Intercepted route for creating a new school via the modal slot.
 *
 * This renders the CreateSchoolDialog when navigating to /schools/new
 * from within the dashboard, preserving the background page.
 */

"use client";

import { useRouter } from "next/navigation";
import { CreateSchoolDialog } from "@/features/school";

export default function SchoolNewPage() {
    const router = useRouter();

    return (
        <CreateSchoolDialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        />
    );
}
