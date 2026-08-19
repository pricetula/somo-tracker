"use client";

import React from "react";
import {
    Combobox,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxList,
} from "@/components/ui/combobox";
import { useAcademicYears } from "@/features/academicyears/hooks";

interface Props {
    className: string;
    withTermSelect: boolean;
}

interface CustomAcademicYear {
    value: string;
    label: string;
    isCurrent: boolean;
    startDate: string;
    endDate: string;
}

interface CustomAcademicTerm {
    label: string;
    value: string;
    termNumber: string;
    startDate: string;
    endDate: string;
    isCurrent: boolean;
    isFinal: boolean;
}

export function AcademicYearHandler({ className, withTermSelect }: Props) {
    const { data } = useAcademicYears();
    const [academicYear, setAcademicYear] = React.useState<CustomAcademicYear | null>(null);
    const [academicTerm, setAcademicTerm] = React.useState<CustomAcademicTerm | null>(null);

    const academicYears = React.useMemo(
        () =>
            data && data?.length
                ? data.map((yr) => ({
                      value: yr.id,
                      label: yr.name,
                      isCurrent: yr.is_current,
                      startDate: yr.start_date,
                      endDate: yr.end_date,
                  }))
                : [],
        [data]
    );

    const mappedAcademicTerms: Map<string, CustomAcademicTerm[]> = React.useMemo(
        () =>
            data && data.length > 0
                ? new Map(
                      data.map((yr) => [
                          yr.id,
                          yr.terms &&
                              yr.terms.map((tr) => ({
                                  label: tr.name,
                                  value: tr.id,
                                  termNumber: tr.term_number,
                                  startDate: tr.start_date,
                                  endDate: tr.end_date,
                                  isCurrent: tr.is_current,
                                  isFinal: tr.is_final,
                              })),
                      ])
                  )
                : new Map(),
        [data]
    );

    const selectedAcademicTerms = React.useMemo(() => {
        if (!academicYear || !mappedAcademicTerms.has(academicYear.value)) return [];
        const t = mappedAcademicTerms.get(academicYear.value);
        return t ?? [];
    }, [mappedAcademicTerms, academicYear]);

    const setCurrentAcademicTerm = React.useCallback(
        (yr: CustomAcademicYear) => {
            if (mappedAcademicTerms.has(yr.value)) {
                const terms = mappedAcademicTerms.get(yr.value);
                const currentTerm = terms && terms.find((tr) => tr.isCurrent);
                if (currentTerm) {
                    setAcademicTerm(currentTerm);
                }
            }
        },
        [mappedAcademicTerms, setAcademicTerm]
    );

    if (academicYears.length && !academicYear) {
        const currentAcademicYear = academicYears.find((yr) => yr.isCurrent);
        if (currentAcademicYear) {
            setAcademicYear(currentAcademicYear);
        }
    }

    if (academicYear && mappedAcademicTerms.has(academicYear.value) && !academicTerm) {
        setCurrentAcademicTerm(academicYear);
    }

    return (
        <section className="max-w-75">
            <div className={className}>
                <Combobox
                    items={academicYears}
                    itemToStringValue={(f: { value: string; label: string }) => f.label}
                    value={academicYear}
                    onValueChange={(d) => {
                        const selectedYear = d as CustomAcademicYear;
                        setAcademicTerm(null);
                        setAcademicYear(selectedYear);
                    }}
                >
                    <ComboboxInput placeholder="Select a academic year" />
                    <ComboboxContent>
                        <ComboboxEmpty>No items found.</ComboboxEmpty>
                        <ComboboxList>
                            {(i) => (
                                <ComboboxItem key={i.value} value={i}>
                                    {i.label}
                                </ComboboxItem>
                            )}
                        </ComboboxList>
                    </ComboboxContent>
                </Combobox>

                {withTermSelect && (
                    <Combobox
                        items={selectedAcademicTerms}
                        itemToStringValue={(f: { value: string; label: string }) => f.label}
                        value={academicTerm}
                        onValueChange={(d) => {
                            const selectedTerm = d as CustomAcademicTerm;
                            setAcademicTerm(selectedTerm);
                        }}
                        disabled={!academicYear}
                    >
                        <ComboboxInput placeholder="Select a academic term" className="max-w-24" />
                        <ComboboxContent>
                            <ComboboxEmpty>No items found.</ComboboxEmpty>
                            <ComboboxList>
                                {(i) => (
                                    <ComboboxItem key={i.value} value={i}>
                                        {i.label}
                                    </ComboboxItem>
                                )}
                            </ComboboxList>
                        </ComboboxContent>
                    </Combobox>
                )}
            </div>
        </section>
    );
}
