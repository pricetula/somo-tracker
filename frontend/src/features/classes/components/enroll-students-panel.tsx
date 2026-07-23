/**
 * EnrollStudentsPanel — Searchable checklist for batch-enrolling students.
 *
 * Fetches students NOT enrolled in the current class, lets the user
 * multi-select via checkboxes, and submits the batch enrollment.
 * Shows inline loading states, error banners, and success feedback.
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Search, Loader2, Check, AlertTriangle } from "lucide-react";

import { getAvailableStudents, batchEnrollStudents } from "@/lib/api/classes";
import { getErrorMessage } from "@/lib/errors";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EnrollStudentsPanelProps {
    classId: string;
    /** Called after successful enrollment to close the overlay. */
    onSuccess?: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EnrollStudentsPanel({ classId, onSuccess }: EnrollStudentsPanelProps) {
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [debouncedSearch, setDebouncedSearch] = useState("");
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [errorBanner, setErrorBanner] = useState<string | null>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);

    // Debounce search input with useEffect + setTimeout
    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search);
        }, 300);
        return () => clearTimeout(timer);
    }, [search]);

    const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(e.target.value);
    }, []);

    // Focus search input on mount
    useEffect(() => {
        searchInputRef.current?.focus();
    }, []);

    // ── Fetch available students ─────────────────────────────────────────
    const {
        data: availableData,
        isLoading,
        isError: isListError,
        error: listError,
    } = useQuery({
        queryKey: ["available-students", classId, debouncedSearch],
        queryFn: () => getAvailableStudents(classId, { search: debouncedSearch, limit: 200 }),
        staleTime: 15_000,
    });

    // ── Batch enrollment mutation ────────────────────────────────────────
    const enrollMutation = useMutation({
        mutationFn: (studentIds: string[]) => batchEnrollStudents(classId, studentIds),
        onSuccess: (data) => {
            toast.success(data.message || `${data.enrolled_count} students successfully enrolled.`);
            // Invalidate class roster and available students queries
            queryClient.invalidateQueries({ queryKey: ["class-roster", classId] });
            queryClient.invalidateQueries({ queryKey: ["available-students", classId] });
            onSuccess?.();
        },
        onError: () => {
            setErrorBanner(
                "Enrollment failed. One or more selected students were updated elsewhere. Please refresh and try again."
            );
        },
    });

    // ── Handlers ─────────────────────────────────────────────────────────
    const handleToggleStudent = useCallback((studentId: string) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(studentId)) {
                next.delete(studentId);
            } else {
                next.add(studentId);
            }
            return next;
        });
        // Clear error banner when user changes selection
        setErrorBanner(null);
    }, []);

    const handleSelectAll = useCallback(() => {
        if (!availableData?.items) return;
        const allIds = new Set(availableData.items.map((s) => s.id));
        setSelectedIds(allIds);
        setErrorBanner(null);
    }, [availableData]);

    const handleDeselectAll = useCallback(() => {
        setSelectedIds(new Set());
        setErrorBanner(null);
    }, []);

    const handleConfirmEnrollment = useCallback(() => {
        if (selectedIds.size === 0) return;
        setErrorBanner(null);
        enrollMutation.mutate(Array.from(selectedIds));
    }, [selectedIds, enrollMutation]);

    // ── Render ───────────────────────────────────────────────────────────
    const students = availableData?.items ?? [];

    return (
        <div className="flex flex-col gap-4">
            {/* Error banner */}
            {errorBanner && (
                <Alert variant="destructive">
                    <AlertTriangle className="h-4 w-4" />
                    <AlertDescription>{errorBanner}</AlertDescription>
                </Alert>
            )}

            {/* Search input */}
            <div className="relative">
                <Search className="text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
                <Input
                    ref={searchInputRef}
                    placeholder="Search by name or admission number..."
                    value={search}
                    onChange={handleSearchChange}
                    className="pl-9"
                />
            </div>

            {/* Selection actions */}
            {students.length > 0 && (
                <div className="text-muted-foreground flex items-center gap-2 text-xs">
                    <button
                        type="button"
                        onClick={handleSelectAll}
                        className="hover:text-foreground hover:underline"
                    >
                        Select all
                    </button>
                    <span aria-hidden>&middot;</span>
                    <button
                        type="button"
                        onClick={handleDeselectAll}
                        className="hover:text-foreground hover:underline"
                    >
                        Deselect all
                    </button>
                    {selectedIds.size > 0 && (
                        <span className="ml-auto tabular-nums">{selectedIds.size} selected</span>
                    )}
                </div>
            )}

            {/* Student list */}
            <div className="max-h-[50vh] overflow-y-auto rounded-md border">
                {isLoading ? (
                    <div className="space-y-2 p-4">
                        <Skeleton className="h-10 w-full" />
                        <Skeleton className="h-10 w-full" />
                        <Skeleton className="h-10 w-3/4" />
                        <Skeleton className="h-10 w-full" />
                        <Skeleton className="h-10 w-5/6" />
                    </div>
                ) : isListError ? (
                    <p className="text-destructive p-4 text-center">{getErrorMessage(listError)}</p>
                ) : students.length === 0 ? (
                    <p className="text-muted-foreground p-6 text-center">
                        {debouncedSearch
                            ? "No students match your search."
                            : "All students are already enrolled in this class."}
                    </p>
                ) : (
                    <ul className="divide-y">
                        {students.map((student) => (
                            <li key={student.id}>
                                <label
                                    className={`hover:bg-muted/50 flex cursor-pointer items-center gap-3 px-4 py-2.5 transition-colors ${
                                        student.current_class_id ? "opacity-50" : ""
                                    }`}
                                >
                                    <Checkbox
                                        checked={selectedIds.has(student.id)}
                                        onCheckedChange={() => handleToggleStudent(student.id)}
                                        disabled={!!student.current_class_id}
                                    />
                                    <div className="flex min-w-0 flex-1 flex-col">
                                        <span className="truncate font-medium">
                                            {student.full_name}
                                        </span>
                                        <span className="text-muted-foreground truncate text-xs">
                                            {student.admission_number &&
                                                `Adm: ${student.admission_number}`}
                                            {student.admission_number &&
                                                student.upi_number &&
                                                " \u00b7 "}
                                            {student.upi_number && `UPI: ${student.upi_number}`}
                                        </span>
                                    </div>
                                    {student.current_class ? (
                                        <span className="text-muted-foreground shrink-0 text-xs">
                                            In: {student.current_class}
                                        </span>
                                    ) : (
                                        <span className="shrink-0 text-xs text-emerald-600">
                                            Unenrolled
                                        </span>
                                    )}
                                </label>
                            </li>
                        ))}
                    </ul>
                )}
            </div>

            {/* Confirm button */}
            <div className="flex items-center justify-end gap-2 pt-2">
                <Button
                    size="sm"
                    onClick={handleConfirmEnrollment}
                    disabled={selectedIds.size === 0 || enrollMutation.isPending}
                >
                    {enrollMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Enrolling...
                        </>
                    ) : (
                        <>
                            <Check className="mr-2 h-4 w-4" />
                            Confirm Enrollment
                        </>
                    )}
                </Button>
            </div>
        </div>
    );
}
