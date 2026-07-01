/**
 * Parents listing page.
 *
 * Shows all parent/guardian profiles with search.
 * Maps to GET /api/v1/parents.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { ParentsTable, useParents, useDeleteParent } from "@/features/parents";

export default function ParentsPage() {
    const router = useRouter();
    const [search, setSearch] = React.useState("");

    // Derive the debounced value without calling setState in an effect
    const debouncedSearch = React.useDeferredValue(search);

    const {
        data: parentsData,
        isLoading: parentsLoading,
        isError: parentsError,
    } = useParents({ search: debouncedSearch || undefined });

    const deleteParent = useDeleteParent();

    const parents = parentsData?.data ?? [];
    const total = parents.length;

    const handleDelete = async (id: string) => {
        if (window.confirm("Delete this parent profile? This cannot be undone.")) {
            deleteParent.mutate(id);
        }
    };

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Parents &amp; Guardians</h1>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {parentsError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load parents. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <ParentsTable
                                parents={parents}
                                total={total}
                                isLoading={parentsLoading}
                                search={search}
                                onSearchChange={setSearch}
                                onDelete={handleDelete}
                                onCreateClick={() => router.push("/parents/new")}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
