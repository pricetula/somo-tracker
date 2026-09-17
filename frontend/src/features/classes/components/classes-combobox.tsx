"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useClasses } from "../hooks/use-classes";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";

interface ClassesComboboxProps {
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
    placeholder?: string;
}

export function ClassesCombobox({
    value,
    onChange,
    disabled,
    placeholder = "Select class",
}: ClassesComboboxProps) {
    const { data, isLoading, isError } = useClasses();

    const items = useMemo(() => {
        const items = data?.items ?? [];
        return items.map((c) => ({
            value: c.id,
            label: `${c.name} — ${c.grade} ${c.stream}`,
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
                    <div className="text-muted-foreground p-4 text-sm">Loading classes...</div>
                )}
                {isError && (
                    <div className="text-destructive p-4 text-sm">Error loading classes</div>
                )}
                {!isLoading && !isError && (
                    <>
                        <ComboboxEmpty>
                            {items.length === 0 ? (
                                <div className="text-muted-foreground space-y-1 p-4 text-sm">
                                    <div>No classes found</div>
                                    <Link href="/classes/add" className="text-primary underline">
                                        Create a class
                                    </Link>
                                </div>
                            ) : (
                                "No classes found"
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
