/**
 * Curriculum Detail Page — full tree view for a single learning area.
 *
 * Shows strands, sub-strands, and performance indicators in a three-tier
 * expandable tree with CRUD actions at every level.
 */

"use client";

import * as React from "react";
import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";

import { CurriculumTree, useLearningAreaTree } from "@/features/curriculum";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export default function CurriculumDetailPage() {
    const params = useParams();
    const id = params.id as string;

    const { data: tree, isLoading, isError } = useLearningAreaTree(id);

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <Button variant="ghost" size="icon-sm" asChild>
                    <Link href="/curriculum">
                        <ArrowLeft className="size-4" />
                        <span className="sr-only">Back</span>
                    </Link>
                </Button>
                <h1 className="text-2xl font-semibold tracking-tight">
                    {isLoading ? (
                        <Skeleton className="inline-block h-7 w-48 align-middle" />
                    ) : (
                        (tree?.name ?? "Learning Area")
                    )}
                </h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    <CurriculumTree tree={tree} isLoading={isLoading} isError={isError} />
                </section>
            </div>
        </div>
    );
}
