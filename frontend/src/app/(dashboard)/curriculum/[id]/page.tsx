/**
 * Curriculum Detail Page — full tree view for a single learning area.
 *
 * Shows strands, sub-strands, and performance indicators in a three-tier
 * expandable tree with CRUD actions at every level.
 */

"use client";

import { useParams } from "next/navigation";

import { CurriculumTree, useLearningAreaTree } from "@/features/curriculum";

export default function CurriculumDetailPage() {
    const params = useParams();
    const id = params.id as string;

    const { data: tree, isLoading, isError } = useLearningAreaTree(id);

    return <CurriculumTree tree={tree} isLoading={isLoading} isError={isError} />;
}
