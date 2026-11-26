"use client";

import { Controller, useFormContext } from "react-hook-form";
import { Label } from "@/components/ui/label";
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import type { EducationSystem } from "../types/education-systems";

interface EducationSystemComboboxProps {
  name: string;
  label?: string;
  disabled?: boolean;
  systems: EducationSystem[];
  placeholder?: string;
}

/**
 * Education system selection combobox wired to react-hook-form via Controller.
 * Parent form provides useForm context.
 */
export function EducationSystemCombobox({
  name,
  label,
  disabled,
  systems,
  placeholder = "Select an education system",
}: EducationSystemComboboxProps) {
  const { control } = useFormContext();

  return (
    <>
      {label && <Label htmlFor={name}>{label}</Label>}
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Combobox
            value={field.value ?? undefined}
            onValueChange={field.onChange}
            disabled={disabled}
          >
            <ComboboxInput placeholder={placeholder} disabled={disabled} />
            <ComboboxContent>
              <ComboboxList>
                {systems.map((system) => (
                  <ComboboxItem key={system.id} value={system.id}>
                    {system.name}
                  </ComboboxItem>
                ))}
              </ComboboxList>
            </ComboboxContent>
          </Combobox>
        )}
      />
    </>
  );
}
