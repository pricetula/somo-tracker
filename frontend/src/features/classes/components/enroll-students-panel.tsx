/**
 * EnrollStudentsPanel — Searchable checklist for batch-enrolling students.
 *
 * Two modes:
 * 1. `academicTermId` is provided (from URL `?academicyear=termId`):
 *    Shows the enrollment form directly, uses that term.
 * 2. `academicTermId` is NOT provided:
 *    Renders academic year + academic term comboboxes first,
 *    then the enrollment form once a term is selected.
 *
 * The selected `academic_term_id` is sent with the POST body so the
 * backend enrolls students into the correct term.
 */

"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Search, Loader2, Check, AlertTriangle } from "lucide-react";

import { getAvailableStudents, batchEnrollStudents } from "@/lib/api/classes";
import { getErrorMessage } from "@/lib/errors";
import { STALE_TIMES } from "@/lib/query-config";
import { useAcademicTerms, AcademicTermCombobox } from "@/features/academic-terms";
import { useAcademicYears, AcademicYearCombobox } from "@/features/academic-years";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { toast } from "sonner";

// ─── Props ─────────────────────────────────────────────────────────────────

interface EnrollStudentsPanelProps {
    classId: string;
    /**
     * Academic term ID. When provided (from URL `?academicyear=termId`),
     * the enrollment form renders directly. When omitted, academic year
     * and term comboboxes are shown for the user to select first.
     */
    academicTermId?: string;
    /** Called after successful enrollment to close the overlay. */
    onSuccess?: () => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function EnrollStudentsPanel({
    classId,
    academicTermId: initialTermId,
    onSuccess,
}: EnrollStudentsPanelProps) {
    const queryClient = useQueryClient();
    const [search, setSearch] = useState("");
    const [debouncedSearch, setDebouncedSearch] = useState("");
    const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
    const [errorBanner, setErrorBanner] = useState<string | null>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);

    // ── Term selection state (used when no initialTermId is given) ────
    const [selectedYearId, setSelectedYearId] = useState("");
    const [selectedTermId, setSelectedTermId] = useState("");

    // If initialTermId is provided, use it directly; otherwise derive from comboboxes
    const resolvedTermId = initialTermId || selectedTermId;

    // Auto-focus search when it appears
    useEffect(() => {
        if (resolvedTermId) {
            // Small delay to let the DOM render
            setTimeout(() => searchInputRef.current?.focus(), 100);
        }
    }, [resolvedTermId]);

    // ── Fetch academic years/terms for combobox mode ─────────────────
    const { data: yearsData } = useAcademicYears();
    const { data: termsData } = useAcademicTerms();

    // ── Debounce search input ────────────────────────────────────────
    useEffect(() => {
        const timer = setTimeout(() => {
            setDebouncedSearch(search);
        }, 300);
        return () => clearTimeout(timer);
    }, [search]);

    const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setSearch(e.target.value);
    }, []);

    // Resolve academic year ID from the selected term or auto-select current year as default
    const currentYearId = yearsData?.find((y) => y.is_current)?.id;
    const yearOrDefault = selectedYearId || (!initialTermId ? currentYearId : undefined) || "";
    const resolvedYearId = initialTermId
        ? termsData?.items?.find((t) => t.id === initialTermId)?.academic_year_id
        : yearOrDefault;

    const {
        data: availableData,
        isLoading,
        isError: isListError,
        error: listError,
    } = useQuery({
        queryKey: ["available-students", classId, debouncedSearch, resolvedYearId, resolvedTermId],
        queryFn: () =>
            getAvailableStudents(classId, {
                search: debouncedSearch,
                limit: 200,
                academic_year_id: resolvedYearId,
                academic_term_id: resolvedTermId,
            }),
        staleTime: STALE_TIMES.LIVE,
        enabled: !!resolvedTermId,
    });

    // ── Batch enrollment mutation ────────────────────────────────────
    const enrollMutation = useMutation({
        mutationFn: (studentIds: string[]) => batchEnrollStudents(classId, studentIds),
        onSuccess: (data) => {
            toast.success(data.message || `${data.enrolled_count} students successfully enrolled.`);
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

    // ── Handlers ─────────────────────────────────────────────────────
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
        setErrorBanner(null);
    }, []);

    const handleSelectAll = useCallback(() => {
        if (!availableData?.items) return;
        setSelectedIds(new Set(availableData.items.map((s) => s.id)));
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

    // ── Render: Term selector (when no initialTermId) ────────────────
    if (!initialTermId) {
        return (
            <div className="flex flex-col gap-4">
                <p className="text-muted-foreground">
                    Select an academic year and term to enroll students into.
                </p>

                {/* Academic Year combobox */}
                <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground font-medium">Academic Year</label>
                    <AcademicYearCombobox
                        value={yearOrDefault}
                        onChange={(v) => {
                            setSelectedYearId(v);
                            setSelectedTermId("");
                            setSelectedIds(new Set());
                        }}
                        placeholder="Select academic year..."
                    />
                </div>

                {/* Academic Term combobox */}
                <div className="flex flex-col gap-1.5">
                    <label className="text-muted-foreground font-medium">Academic Term</label>
                    <AcademicTermCombobox
                        value={selectedTermId}
                        onChange={(v) => {
                            setSelectedTermId(v);
                            setSelectedIds(new Set());
                            setErrorBanner(null);
                        }}
                        placeholder="Select academic term..."
                    />
                </div>

                {/* Enrollment form — only when a term is selected */}
                {resolvedTermId && (
                    <>
                        <hr className="my-2" />
                        {renderEnrollmentForm()}
                    </>
                )}
            </div>
        );
    }

    // ── Render: Direct enrollment form (when term is in URL) ─────────
    return <div className="flex flex-col gap-4">{renderEnrollmentForm()}</div>;

    // ── Shared enrollment form ────────────────────────────────────────
    function renderEnrollmentForm() {
        const students = availableData?.items ?? [];

        return (
            <>
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
                    <div className="text-muted-foreground flex items-center gap-2">
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
                            <span className="ml-auto tabular-nums">
                                {selectedIds.size} selected
                            </span>
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
                        <p className="text-destructive p-4 text-center">
                            {getErrorMessage(listError)}
                        </p>
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
                                            <span className="text-muted-foreground truncate">
                                                {student.admission_number &&
                                                    `Adm: ${student.admission_number}`}
                                                {student.admission_number &&
                                                    student.upi_number &&
                                                    " \u00b7 "}
                                                {student.upi_number && `UPI: ${student.upi_number}`}
                                            </span>
                                        </div>
                                        {student.current_class ? (
                                            <span className="text-muted-foreground shrink-0">
                                                In: {student.current_class}
                                            </span>
                                        ) : (
                                            <span className="shrink-0 text-emerald-600">
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
            </>
        );
    }
}
