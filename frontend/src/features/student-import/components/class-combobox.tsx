"use client";

import * as React from "react";
import { ComboboxInput } from "./combobox-input";
import type { ClassesMap } from "../types";

interface ClassComboboxProps {
    value: string;
    classesMap: ClassesMap;
    onChange: (value: string) => void;
}

export function ClassCombobox({ value, classesMap, onChange }: ClassComboboxProps) {
    const classOptions = React.useMemo(() => {
        return Array.from(classesMap.entries()).map(([key, c]) => ({
            key,
            label: c.name,
        }));
    }, [classesMap]);

    return (
        <ComboboxInput
            placeholder="Class"
            value={value}
            options={classOptions}
            onChange={onChange}
        />
    );
}
