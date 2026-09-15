"use client";

import { useState, useMemo } from "react";
import { useStreams } from "../hooks/use-streams";
import { useCreateStreams } from "../hooks/use-streams";
import {
    Combobox,
    ComboboxInput,
    ComboboxContent,
    ComboboxList,
    ComboboxItem,
    ComboboxEmpty,
} from "@/components/ui/combobox";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

interface StreamsComboboxProps {
    value?: string;
    onChange: (value: string) => void;
    disabled?: boolean;
}

export function StreamsCombobox({ value, onChange, disabled }: StreamsComboboxProps) {
    const { data: streams, isLoading, isError } = useStreams();
    const { mutate: createStreams, isPending } = useCreateStreams();
    const [createName, setCreateName] = useState("");
    const [showCreate, setShowCreate] = useState(false);

    const items = useMemo(() => {
        if (!streams) return [];
        return streams.map((s) => ({
            value: s.id,
            label: s.name,
        }));
    }, [streams]);

    const selectedItem = items.find((i) => i.value === value);

    const handleCreate = () => {
        const trimmed = createName.trim();
        if (!trimmed) return;
        createStreams([trimmed], {
            onSuccess: (data) => {
                if (data.stream_ids.length > 0) {
                    onChange(data.stream_ids[0]);
                    setCreateName("");
                    setShowCreate(false);
                }
            },
        });
    };

    return (
        <div className="space-y-2">
            <Combobox
                items={items}
                itemToStringValue={(item) => item?.label ?? ""}
                value={selectedItem ?? null}
                onValueChange={(item) => onChange(item?.value ?? "")}
                disabled={disabled || isPending}
            >
                <ComboboxInput placeholder="Select or search stream" showClear />
                <ComboboxContent>
                    {isLoading && (
                        <div className="text-muted-foreground p-4 text-sm">Loading streams...</div>
                    )}
                    {isError && (
                        <div className="text-destructive p-4 text-sm">Error loading streams</div>
                    )}
                    {!isLoading && !isError && (
                        <>
                            <ComboboxEmpty>No streams found</ComboboxEmpty>
                            <ComboboxList>
                                {(item) => (
                                    <ComboboxItem key={item.value} value={item}>
                                        {item.label}
                                    </ComboboxItem>
                                )}
                            </ComboboxList>
                            {items.length > 0 && (
                                <>
                                    <hr className="border-border/50 my-1" />
                                    <button
                                        type="button"
                                        className="text-muted-foreground hover:text-foreground w-full cursor-pointer px-2 py-1 text-left text-xs"
                                        onClick={() => setShowCreate((v) => !v)}
                                    >
                                        {showCreate ? "Cancel create" : "Create new stream..."}
                                    </button>
                                </>
                            )}
                        </>
                    )}
                    {showCreate && (
                        <div className="space-y-2 p-2">
                            <Input
                                placeholder="Stream name"
                                value={createName}
                                onChange={(e) => setCreateName(e.target.value)}
                                autoFocus
                                className="h-7 text-xs"
                            />
                            <div className="flex gap-2">
                                <Button
                                    size="sm"
                                    variant="default"
                                    onClick={handleCreate}
                                    disabled={isPending || !createName.trim()}
                                    className="h-7 text-xs"
                                >
                                    <Plus className="size-3" />
                                    {isPending ? "Creating..." : "Create"}
                                </Button>
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => {
                                        setShowCreate(false);
                                        setCreateName("");
                                    }}
                                    className="h-7 text-xs"
                                >
                                    Cancel
                                </Button>
                            </div>
                        </div>
                    )}
                </ComboboxContent>
            </Combobox>
        </div>
    );
}
