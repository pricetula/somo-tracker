"use client";

import { useMemo } from "react";
import { useTeachers } from "../hooks/use-teachers-list";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";

interface TeachersComboboxProps {
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
    placeholder?: string;
}

export function TeachersCombobox({
    value,
    onChange,
    disabled,
    placeholder = "Select teacher",
}: TeachersComboboxProps) {
    const { data, isLoading, isError } = useTeachers();

    const items = useMemo(() => {
        const items = data?.items ?? [];
        return items.map((t) => ({
            value: t.membership_id,
            label: `${t.full_name} (${t.email})`,
        }));
    }, [data]);

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
                    <div className="text-muted-foreground p-4 text-sm">Loading teachers...</div>
                )}
                {isError && (
                    <div className="text-destructive p-4 text-sm">Error loading teachers</div>
                )}
                {!isLoading && !isError && (
                    <>
                        <ComboboxEmpty>No teachers found</ComboboxEmpty>
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
