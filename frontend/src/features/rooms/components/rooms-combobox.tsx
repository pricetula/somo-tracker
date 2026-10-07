"use client";

import { useMemo } from "react";
import { useRooms } from "../hooks/use-rooms";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";

interface RoomsComboboxProps {
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
    placeholder?: string;
}

export function RoomsCombobox({
    value,
    onChange,
    disabled,
    placeholder = "Select room",
}: RoomsComboboxProps) {
    const { data, isLoading, isError } = useRooms();

    const items = useMemo(() => {
        const items = data?.items ?? [];
        return items.map((r) => ({
            value: r.id,
            label: `${r.name}${r.capacity ? ` • ${r.capacity} seats` : ""}`,
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
                {isLoading && <div className="text-muted-foreground p-4">Loading rooms...</div>}
                {isError && <div className="text-destructive p-4">Error loading rooms</div>}
                {!isLoading && !isError && (
                    <>
                        <ComboboxEmpty>No rooms found</ComboboxEmpty>
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
