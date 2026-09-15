"use client";

import { useMemo } from "react";
import { useGrades } from "../hooks/use-grades";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";

interface GradesComboboxProps {
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
    placeholder?: string;
}

export function GradesCombobox({
    value,
    onChange,
    disabled,
    placeholder = "Select grade",
}: GradesComboboxProps) {
    const { data: grades, isLoading, isError } = useGrades();

    const items = useMemo(() => {
        if (!grades) return [];
        return grades.map((g) => ({
            value: g.id,
            label: g.local_label,
        }));
    }, [grades]);

    const selectedItem = items.find((i) => i.value === value);

    return (
        <Combobox
            items={items}
            itemToStringValue={(item) => item?.label ?? ""}
            value={selectedItem ?? null}
            onValueChange={(item) => onChange(item?.value ?? "")}
            disabled={disabled}
        >
            <ComboboxInput placeholder={placeholder} showClear />
            <ComboboxContent>
                {isLoading && (
                    <div className="text-muted-foreground p-4 text-sm">Loading grades...</div>
                )}
                {isError && (
                    <div className="text-destructive p-4 text-sm">Error loading grades</div>
                )}
                {!isLoading && !isError && (
                    <>
                        <ComboboxEmpty>No grades found</ComboboxEmpty>
                        <ComboboxList>
                            {(item) => (
                                <ComboboxItem key={item.value} value={item}>
                                    {item.label}
                                </ComboboxItem>
                            )}
                        </ComboboxList>
                    </>
                )}
            </ComboboxContent>
        </Combobox>
    );
}
