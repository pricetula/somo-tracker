"use client";

import { useMemo } from "react";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";

// ---- YearSelectProps props for the component ----
interface YearSelectProps {
    year: number;
    onSelect: (year: number) => void;
}

export function YearSelect({ year, onSelect }: YearSelectProps) {
    const currentYear = useMemo(() => new Date().getFullYear(), []);

    const yearOptions = useMemo(
        () => Array.from({ length: 14 }, (_, i) => currentYear - i),
        [currentYear]
    );

    return (
        <Select value={year} onValueChange={(v) => v && onSelect(v)}>
            <SelectTrigger className="w-full">
                <SelectValue placeholder="Select a year" />
            </SelectTrigger>
            <SelectContent>
                {yearOptions.map((y) => (
                    <SelectItem key={y} value={y}>
                        {y}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    );
}
