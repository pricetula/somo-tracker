/**
 * Term Selection Step — the user picks the academic year and term
 * for this student import batch. Both are queried from the backend
 * via combobox (Popover + Command) with search filtering.
 *
 * Internally uses backend IDs for the combobox but maps to the year/term
 * name strings expected by the wizard state and submit API.
 */

"use client";

import * as React from "react";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { AcademicCombobox, type ComboboxOption } from "./academic-combobox";
import { useAcademicYears, useAcademicPeriods } from "../hooks/use-academic-periods";

export interface TermSelectionStepProps {
    academicYear: string;
    term: string;
    onAcademicYearChange: (year: string) => void;
    onTermChange: (term: string) => void;
    onContinue: () => void;
    onBack?: () => void;
}

export function TermSelectionStep({
    academicYear,
    term,
    onAcademicYearChange,
    onTermChange,
    onContinue,
    onBack,
}: TermSelectionStepProps) {
    const { years, yearsLoading, yearsError, retryYears } = useAcademicYears();
    const { periods, periodsLoading, periodsError } = useAcademicPeriods(
        // Resolve the selected year's ID from the current academicYear name
        years?.find?.((y) => y.name === academicYear)?.id ?? null
    );

    // Auto-select current year on load (maps name → name)
    React.useEffect(() => {
        if (!academicYear && years.length > 0 && !yearsLoading) {
            const current = years.find((y) => y.is_current) ?? years[0];
            onAcademicYearChange(current.name);
        }
    }, [years, yearsLoading, academicYear, onAcademicYearChange]);

    // Auto-select current period when periods load
    React.useEffect(() => {
        if (!term && periods.length > 0 && !periodsLoading) {
            const current = periods.find((p) => p.is_current) ?? periods[0];
            onTermChange(current.name);
        }
    }, [periods, periodsLoading, term, onTermChange]);

    // Build combobox options using year name as value (matches wizard state)
    const yearOptions: ComboboxOption[] = React.useMemo(
        () =>
            years.map((y) => ({
                value: y.name,
                label: y.name,
                isCurrent: y.is_current,
            })),
        [years]
    );

    const periodOptions: ComboboxOption[] = React.useMemo(
        () =>
            periods.map((p) => ({
                value: p.name,
                label: p.name,
                isCurrent: p.is_current,
            })),
        [periods]
    );

    return (
        <div className="space-y-6">
            <div>
                <p className="text-muted-foreground text-sm">
                    Select the academic year and period for this student import batch. The current
                    period is pre-selected by default.
                </p>
            </div>

            {/* Error state */}
            {yearsError && (
                <div className="bg-muted/30 flex items-center gap-2 rounded-sm px-3 py-2">
                    <p className="text-destructive text-xs">Failed to load academic years.</p>
                    <Button
                        variant="ghost"
                        size="sm"
                        className="h-auto px-2 py-0.5 text-xs"
                        onClick={retryYears}
                    >
                        Retry
                    </Button>
                </div>
            )}

            <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
                {/* Academic Year */}
                <div className="space-y-2">
                    <Label htmlFor="academic-year">Academic Year</Label>
                    <AcademicCombobox
                        options={yearOptions}
                        value={academicYear}
                        onValueChange={onAcademicYearChange}
                        placeholder="Select year..."
                        searchPlaceholder="Search year..."
                        emptyText="No years found."
                        loading={yearsLoading}
                        disabled={!!yearsError}
                    />
                </div>

                {/* Academic Period */}
                <div className="space-y-2">
                    <Label htmlFor="academic-period">Academic Period</Label>
                    <AcademicCombobox
                        options={periodOptions}
                        value={term}
                        onValueChange={onTermChange}
                        placeholder={academicYear ? "Select period..." : "Select a year first..."}
                        searchPlaceholder="Search period..."
                        emptyText={periodsError ? "Failed to load periods." : "No periods found."}
                        loading={periodsLoading}
                        disabled={!academicYear || !!periodsError}
                    />
                </div>
            </div>

            {/* Current period indicator */}
            {academicYear && term && (
                <div className="bg-muted/30 rounded-sm px-3 py-2">
                    <p className="text-muted-foreground text-xs">
                        Selected period:{" "}
                        <span className="text-foreground font-medium">
                            {academicYear} — {term}
                        </span>
                    </p>
                </div>
            )}

            {/* Actions */}
            <div className="flex items-center justify-between">
                {onBack ? (
                    <Button variant="ghost" onClick={onBack}>
                        Back
                    </Button>
                ) : (
                    <div />
                )}
                <Button onClick={onContinue} disabled={!academicYear || !term || !!yearsError}>
                    Continue
                </Button>
            </div>
        </div>
    );
}
