/**
 * ClassListPage — Renders the classes directory with search, Linear-style
 * filter dropdown, and a virtualized table.
 */

"use client";

import * as React from "react";

import { ClassTable } from "./class-table";
import { ClassFilterDropdown } from "./class-filter-dropdown";
import { useClassList, useGrades } from "@/features/classes/hooks/use-classes";

// ─── State shape ───────────────────────────────────────────────────────────

interface FilterState {
    gradeIds: string[];
    isActive: boolean | null;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function ClassListPage() {
    // ── Search state ──────────────────────────────────────────────────────
    const [search, setSearch] = React.useState("");
    const [debouncedSearch, setDebouncedSearch] = React.useState("");

    // ── Filter state ──────────────────────────────────────────────────────
    const [filters, setFilters] = React.useState<FilterState>({
        gradeIds: [],
        isActive: null,
    });

    // Debounce search input
    React.useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search);
        }, 300);
        return () => clearTimeout(timer);
    }, [search]);

    // ── Data ──────────────────────────────────────────────────────────────
    const { data: grades = [] } = useGrades();
    const { data: classes, isLoading } = useClassList({
        search: debouncedSearch || undefined,
        grade_ids: filters.gradeIds.length > 0 ? filters.gradeIds : undefined,
        is_active: filters.isActive ?? undefined,
    });

    const classList = classes ?? [];

    // ── Empty state check ──────────────────────────────────────────────────
    const isEmpty =
        !isLoading &&
        classList.length === 0 &&
        !debouncedSearch &&
        filters.gradeIds.length === 0 &&
        filters.isActive === null;

    return (
        <div className="flex flex-1 flex-col">
            {/* ── Page Header ─────────────────────────────────────────── */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Classes</h1>
                {classList.length > 0 && (
                    <span className="border-border/40 text-muted-foreground inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium">
                        {classList.length}
                    </span>
                )}
            </div>

            {/* ── Content ──────────────────────────────────────────────── */}
            {isEmpty ? (
                <div className="flex flex-1 items-center justify-center p-8">
                    <div className="text-center">
                        <h3 className="text-lg font-medium">No classes yet</h3>
                        <p className="text-muted-foreground mt-1 text-sm">
                            Generate classes using the onboarding flow.
                        </p>
                    </div>
                </div>
            ) : (
                <ClassTable
                    classes={classList}
                    search={search}
                    onSearchChange={setSearch}
                    isLoading={isLoading}
                    filterSlot={
                        <ClassFilterDropdown
                            grades={grades}
                            filters={filters}
                            onFiltersChange={setFilters}
                        />
                    }
                />
            )}
        </div>
    );
}
