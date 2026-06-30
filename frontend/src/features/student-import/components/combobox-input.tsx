"use client";

import * as React from "react";

export interface ComboboxOption {
    key: string;
    label: string;
}

interface ComboboxInputProps {
    placeholder: string;
    value: string;
    options: ComboboxOption[];
    onChange: (value: string) => void;
}

export function ComboboxInput({ placeholder, value, options, onChange }: ComboboxInputProps) {
    const [open, setOpen] = React.useState(false);
    const [search, setSearch] = React.useState("");
    const inputRef = React.useRef<HTMLInputElement>(null);
    const listRef = React.useRef<HTMLDivElement>(null);

    const filtered = React.useMemo(() => {
        if (!search) return options;
        const q = search.toLowerCase();
        return options.filter((o) => o.label.toLowerCase().includes(q));
    }, [options, search]);

    const selectedLabel = options.find((o) => o.key === value)?.label ?? value;

    // Close on outside click
    React.useEffect(() => {
        if (!open) return;
        function handleClick(e: MouseEvent) {
            if (
                inputRef.current &&
                !inputRef.current.contains(e.target as Node) &&
                listRef.current &&
                !listRef.current.contains(e.target as Node)
            ) {
                setOpen(false);
            }
        }
        document.addEventListener("mousedown", handleClick);
        return () => document.removeEventListener("mousedown", handleClick);
    }, [open]);

    return (
        <div className="relative">
            <input
                ref={inputRef}
                placeholder={placeholder}
                value={open ? search : selectedLabel}
                onChange={(e) => {
                    setSearch(e.target.value);
                    if (!open) setOpen(true);
                }}
                onFocus={() => {
                    setOpen(true);
                    setSearch("");
                }}
                className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex h-9 w-full rounded-md border px-3 py-1 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
            />
            {open && filtered.length > 0 && (
                <div
                    ref={listRef}
                    className="bg-popover text-popover-foreground absolute top-full left-0 z-50 mt-1 max-h-48 w-full overflow-auto rounded-md shadow-lg"
                >
                    {filtered.map((opt) => (
                        <button
                            key={opt.key}
                            onClick={() => {
                                onChange(opt.key);
                                setOpen(false);
                                setSearch("");
                            }}
                            className="hover:bg-accent hover:text-accent-foreground w-full px-3 py-1.5 text-left text-sm"
                        >
                            {opt.label}
                        </button>
                    ))}
                </div>
            )}
            {open && filtered.length === 0 && (
                <div className="bg-popover text-popover-foreground absolute top-full left-0 z-50 mt-1 w-full rounded-md px-3 py-2 text-xs shadow-lg">
                    No matches
                </div>
            )}
        </div>
    );
}
