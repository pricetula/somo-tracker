"use client";

import { Controller, useFormContext } from "react-hook-form";
import { Label } from "@/components/ui/label";
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
} from "@/components/ui/combobox";
import type { Country } from "../types/countries";

interface CountryComboboxProps {
  name: string;
  label?: string;
  disabled?: boolean;
  countries: Country[];
  placeholder?: string;
}

/**
 * Country selection combobox wired to react-hook-form via Controller.
 * Parent form provides useForm context.
 */
export function CountryCombobox({
  name,
  label,
  disabled,
  countries,
  placeholder = "Select a country",
}: CountryComboboxProps) {
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
            <ComboboxTrigger>
              <ComboboxInput placeholder={placeholder} disabled={disabled} />
            </ComboboxTrigger>
            <ComboboxContent>
              <ComboboxList>
                {countries.map((country) => (
                  <ComboboxItem key={country.id} value={country.id}>
                    {country.name}
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
