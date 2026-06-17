/**
 * Students Directory Page
 *
 * Hyper-minimalist, borderless design. Full-width virtual table canvas.
 * No card wrappers — only whitespace, typography, and crisp states.
 */

"use client";

import * as React from "react";

import {
    StudentTable,
    StudentSheet,
    ImportModal,
    EmptyState,
    useStudents,
} from "@/features/students";

export default function StudentsPage() {
    // ── State ────────────────────────────────────────────────────────────
    const [page, setPage] = React.useState(1);
    const [search, setSearch] = React.useState("");
    const [debouncedSearch, setDebouncedSearch] = React.useState("");
    const [sheetOpen, setSheetOpen] = React.useState(false);
    const [importOpen, setImportOpen] = React.useState(false);

    // Debounce search input
    React.useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search);
            setPage(1); // Reset to page 1 on new search
        }, 300);
        return () => clearTimeout(timer);
    }, [search]);

    // ── Data ─────────────────────────────────────────────────────────────
    const { data, isLoading } = useStudents({
        page,
        per_page: 50,
        search: debouncedSearch,
    });

    const students = data?.students ?? [];
    const total = data?.total ?? 0;

    // ── Derived state ────────────────────────────────────────────────────
    const isEmpty = !isLoading && total === 0 && !debouncedSearch;

    return (
        <div className="flex flex-1 flex-col">
            {/* ── Page Header ─────────────────────────────────────────── */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Students</h1>
                {total > 0 && (
                    <span className="border-border/40 text-muted-foreground inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium">
                        {total}
                    </span>
                )}
            </div>

            {/* ── Content ──────────────────────────────────────────────── */}
            {isEmpty ? (
                <EmptyState
                    onManualAdd={() => setSheetOpen(true)}
                    onUploadCSV={() => setImportOpen(true)}
                />
            ) : (
                <StudentTable
                    students={students}
                    total={total}
                    search={search}
                    onSearchChange={setSearch}
                    onAddStudent={() => setSheetOpen(true)}
                    onUploadCSV={() => setImportOpen(true)}
                    isLoading={isLoading}
                />
            )}

            {/* ── Add Student Sheet ────────────────────────────────────── */}
            <StudentSheet open={sheetOpen} onOpenChange={setSheetOpen} />

            {/* ── Import CSV Modal ─────────────────────────────────────── */}
            <ImportModal open={importOpen} onOpenChange={setImportOpen} />
        </div>
    );
}
