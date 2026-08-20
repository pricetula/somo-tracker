"use client";

import { Label } from "@/components/ui/label";

const COLOR_OPTIONS = [
    "#ef4444",
    "#f97316",
    "#eab308",
    "#22c55e",
    "#14b8a6",
    "#3b82f6",
    "#8b5cf6",
    "#ec4899",
    "#78716c",
    "#06b6d4",
];

export function ColorPicker({
    value,
    onChange,
}: {
    value: string;
    onChange: (color: string) => void;
}) {
    return (
        <div className="space-y-2">
            <Label>Colour</Label>
            <div className="flex flex-wrap gap-2">
                {COLOR_OPTIONS.map((c) => (
                    <button
                        key={c}
                        type="button"
                        onClick={() => onChange(c)}
                        className={`h-7 w-7 rounded-full border-2 transition-all ${
                            value === c
                                ? "border-foreground scale-110"
                                : "border-transparent hover:scale-110"
                        }`}
                        style={{ backgroundColor: c }}
                        aria-label={`Select colour ${c}`}
                    />
                ))}
                {/* Custom color input */}
                <label
                    className="border-muted-foreground/50 text-muted-foreground hover:border-foreground/50 flex h-7 w-7 cursor-pointer items-center justify-center rounded-full border-2 border-dashed text-xs"
                    aria-label="Custom colour"
                >
                    <span className="text-lg leading-none">+</span>
                    <input
                        type="color"
                        value={value && !COLOR_OPTIONS.includes(value) ? value : "#000000"}
                        onChange={(e) => onChange(e.target.value)}
                        className="sr-only"
                    />
                </label>
            </div>
        </div>
    );
}
