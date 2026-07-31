"use client";

export function ColorDot({ color }: { color: string }) {
    if (!color) return null;
    return (
        <span
            className="inline-block h-3.5 w-3.5 rounded-full"
            style={{ backgroundColor: color }}
            aria-hidden="true"
        />
    );
}
