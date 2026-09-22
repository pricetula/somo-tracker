"use client";

import { useMemo } from "react";
import Link from "next/link";
import { useStreams } from "../hooks/use-streams";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";

interface StreamsComboboxProps {
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
}

export function StreamsCombobox({ value, onChange, disabled }: StreamsComboboxProps) {
    const { data: streams, isLoading, isError } = useStreams();

    const items = useMemo(() => {
        if (!streams) return [];
        return streams.map((s) => ({
            value: s.id,
            label: s.name,
        }));
    }, [streams]);

    const selectedItem = items.find((i) => i.value === value);

    return (
        <div className="space-y-2">
            <Combobox
                items={items}
                itemToStringValue={(item) => item?.label ?? ""}
                value={selectedItem ?? null}
                onValueChange={(item) => onChange(item?.value ?? "")}
                disabled={disabled}
            >
                <ComboboxInput placeholder="Select stream" showClear />
                <ComboboxContent>
                    {isLoading && (
                        <div className="text-muted-foreground p-4">Loading streams...</div>
                    )}
                    {isError && <div className="text-destructive p-4">Error loading streams</div>}
                    {!isLoading && !isError && (
                        <>
                            <ComboboxEmpty>
                                {items.length === 0 ? (
                                    <div className="text-muted-foreground space-y-1 p-4">
                                        <div>No streams found</div>
                                        <Link
                                            href="/settings/streams/add"
                                            className="text-primary underline"
                                        >
                                            Add a stream
                                        </Link>
                                    </div>
                                ) : (
                                    "No streams found"
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
        </div>
    );
}
