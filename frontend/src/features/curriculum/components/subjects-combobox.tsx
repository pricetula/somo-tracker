"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useSubjects } from "../hooks/use-subjects";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";

interface SubjectsComboboxProps {
    gradeId?: string;
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
    placeholder?: string;
}

export function SubjectsCombobox({
    gradeId,
    value,
    onChange,
    disabled,
    placeholder = "Select subject",
}: SubjectsComboboxProps) {
    const { data, isLoading, isError } = useSubjects();

    const items = useMemo(() => {
        let items = (data?.items ?? []).map((s) => ({
            value: s.id,
            label: `${s.name} (${s.code})`,
            data: s,
        }));
        if (gradeId) {
            items = items.filter((i) => i.data.gradeId === gradeId);
        }
        return items;
    }, [data, gradeId]);

    const selectedItem = items.find((i) => i.value === value) ?? null;

    return (
        <Combobox
            items={items}
            itemToStringValue={(item) => item?.label ?? ""}
            value={selectedItem}
            onValueChange={(item) => onChange(item?.value ?? "")}
            disabled={disabled}
        >
            <ComboboxInput placeholder={placeholder} showClear />
            <ComboboxContent>
                {isLoading && (
                    <div className="text-muted-foreground p-4 text-sm">Loading subjects...</div>
                )}
                {isError && (
                    <div className="text-destructive p-4 text-sm">Error loading subjects</div>
                )}
                {!isLoading && !isError && (
                    <>
                        <ComboboxEmpty>
                            {items.length === 0 ? (
                                <div className="text-muted-foreground space-y-1 p-4 text-sm">
                                    <div>No subjects found</div>
                                    <Link href="/curriculum/add" className="text-primary underline">
                                        Add a subject
                                    </Link>
                                </div>
                            ) : (
                                "No subjects found"
                            )}
                        </ComboboxEmpty>
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
