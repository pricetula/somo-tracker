"use client";

import * as React from "react";
import { ComboboxInput } from "./combobox-input";
import type { ParentsMap } from "../types";

interface ParentComboboxProps {
    value: string;
    parentsMap: ParentsMap;
    onChange: (value: string) => void;
}

export function ParentCombobox({ value, parentsMap, onChange }: ParentComboboxProps) {
    const parentOptions = React.useMemo(() => {
        return Array.from(parentsMap.entries()).map(([key, p]) => ({
            key,
            label: p.full_name,
        }));
    }, [parentsMap]);

    return (
        <ComboboxInput
            placeholder="Parent"
            value={value}
            options={parentOptions}
            onChange={onChange}
        />
    );
}
