/**
 * Parent Detail page.
 *
 * Shows parent info and linked students with link/unlink actions.
 * Maps to GET /api/v1/parents/:id.
 */

"use client";

import { useParams } from "next/navigation";

import { ParentDetailView } from "@/features/parents";

export default function ParentDetailPage() {
    const params = useParams();
    const id = params.id as string;

    return <ParentDetailView parentId={id} />;
}
