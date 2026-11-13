'use client';

import { Label } from '@/components/ui/label';
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
} from '@/components/ui/combobox';
import { useCountries } from '../hooks/use-countries';

interface CountryComboboxProps {
  value: string | null;
  onChange: (countryId: string | null) => void;
  disabled?: boolean;
}

export function CountryCombobox({
  value,
  onChange,
  disabled = false,
}: CountryComboboxProps) {
  const { data: countries = [], isLoading } = useCountries();

  return (
    <Combobox
      value={value ?? undefined}
      onValueChange={(countryId) => onChange(countryId ?? null)}
      disabled={disabled}
    >
      <Label htmlFor="country-combobox">Country</Label>
      <ComboboxInput
        id="country-combobox"
        placeholder="Select a country..."
        disabled={disabled}
      />
      <ComboboxTrigger />
      <ComboboxContent>
        <ComboboxList>
          {isLoading && (
            <ComboboxEmpty>Loading...</ComboboxEmpty>
          )}
          {!isLoading && countries.length === 0 && (
            <ComboboxEmpty>No countries found</ComboboxEmpty>
          )}
          {!isLoading &&
            countries.map((country) => (
              <ComboboxItem key={country.id} value={country.id}>
                {country.country_name} ({country.country_code})
              </ComboboxItem>
            ))}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
