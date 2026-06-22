/**
 * Inline Input component with error state for the staff import forms.
 *
 * A lightweight wrapper around a native `<input>` styled to match shadcn's
 * Input component, with support for an `error` visual state.
 */

"use client";

import * as React from "react";

interface ImportInputProps {
    placeholder?: string;
    value: string;
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
    className?: string;
    error?: boolean;
}

export function ImportInput({ placeholder, value, onChange, className = "" }: ImportInputProps) {
    return (
        <input
            type="text"
            placeholder={placeholder}
            value={value}
            onChange={onChange}
            className={`border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring h-9 w-full rounded-md border px-3 text-sm transition-colors focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 ${className}`}
        />
    );
}
