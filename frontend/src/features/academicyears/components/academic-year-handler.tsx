"use client";

import React from "react";
import { parseISO } from "date-fns";
import { CalendarCheck2, CalendarMinus, CalendarClock } from "lucide-react";
import {
    Combobox,
    ComboboxContent,
    ComboboxEmpty,
    ComboboxInput,
    ComboboxItem,
    ComboboxList,
} from "@/components/ui/combobox";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Progress } from "@/components/ui/progress";
import { useAcademicYears } from "@/features/academicyears/hooks";
import { formatDate } from "@/lib/format-date";

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

const currentTime = new Date().getTime();

export function AcademicYearHandler() {
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

    const selectedAcademicTermPercentageProgress = React.useMemo(() => {
        const startTime = academicTerm?.startDate ? parseISO(academicTerm.startDate).getTime() : 0;
        const endTime = academicTerm?.endDate ? parseISO(academicTerm.endDate).getTime() : 0;
        const progress = ((currentTime - startTime) / (endTime - startTime)) * 100;
        if (progress < 0 || !Number.isFinite(progress)) return 0;
        if (progress > 100) return 100;
        return progress;
    }, [academicTerm]);

    const setCurrentTermBasedOnAcademicYear = React.useCallback(
        (yr: CustomAcademicYear) => {
            if (yr && mappedAcademicTerms.has(yr.value)) {
                const terms = mappedAcademicTerms.get(yr.value);
                const currentAcademicTerm = terms && terms.find((tr) => tr.isCurrent);
                if (currentAcademicTerm) {
                    setAcademicTerm(currentAcademicTerm);
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
        setCurrentTermBasedOnAcademicYear(academicYear);
    }

    return (
        <section className="max-w-sm">
            <div className="mb-8 flex gap-4">
                <Combobox
                    items={academicYears}
                    itemToStringValue={(f: { value: string; label: string }) => f.label}
                    value={academicYear}
                    onValueChange={(d) => {
                        setAcademicYear(d as CustomAcademicYear);
                        setCurrentTermBasedOnAcademicYear(d as CustomAcademicYear);
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

                <Combobox
                    items={selectedAcademicTerms}
                    itemToStringValue={(f: { value: string; label: string }) => f.label}
                    value={academicTerm}
                    onValueChange={(d) => {
                        setAcademicTerm(d as CustomAcademicTerm);
                    }}
                >
                    <ComboboxInput placeholder="Select a academic term" />
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
            </div>

            <Tooltip>
                <TooltipTrigger asChild>
                    <div className="flex items-center gap-4">
                        <Progress
                            value={selectedAcademicTermPercentageProgress}
                            className="max-w-[90%] [&>div]:bg-green-500"
                        />
                        {(selectedAcademicTermPercentageProgress === 100 && (
                            <CalendarCheck2 className="text-green-500" size={16} />
                        )) ||
                            (selectedAcademicTermPercentageProgress === 0 && (
                                <CalendarMinus className="text-muted-foreground" size={16} />
                            )) || <CalendarClock className="text-blue-500" size={16} />}
                    </div>
                </TooltipTrigger>
                <TooltipContent side="bottom">
                    <div className="text-sm">
                        <span className="text-muted-foreground">Academic term starts on</span>
                        <span className="mx-1">{formatDate(academicTerm?.startDate || "")}</span>
                        <span className="text-muted-foreground">to</span>
                        <span className="ml-1">{formatDate(academicTerm?.endDate || "")}</span>
                    </div>
                </TooltipContent>
            </Tooltip>
        </section>
    );
}
