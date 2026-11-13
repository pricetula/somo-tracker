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
import { useEducationSystems } from '../hooks/use-education-systems';

interface EducationSystemComboboxProps {
  value: string | null;
  countryId: string | null;
  onChange: (educationSystemId: string | null) => void;
  disabled?: boolean;
}

export function EducationSystemCombobox({
  value,
  countryId,
  onChange,
  disabled = false,
}: EducationSystemComboboxProps) {
  const { data: systems = [], isLoading } = useEducationSystems(countryId);
  const isDisabled = disabled || !countryId;

  return (
    <Combobox
      value={value ?? undefined}
      onValueChange={(systemId) => onChange(systemId ?? null)}
      disabled={isDisabled}
    >
      {!countryId ? (
        <p className="text-muted-foreground">
          Select a country first to view education systems.
        </p>
      ) : (
        <>
          <Label htmlFor="education-system-combobox">Education System</Label>
          <ComboboxInput
            id="education-system-combobox"
            placeholder="Select an education system..."
            disabled={isDisabled}
          />
          <ComboboxTrigger />
          <ComboboxContent>
            <ComboboxList>
              {isLoading && <ComboboxEmpty>Loading...</ComboboxEmpty>}
              {!isLoading && systems.length === 0 && (
                <ComboboxEmpty>No education systems found</ComboboxEmpty>
              )}
              {!isLoading &&
                systems.map((system) => (
                  <ComboboxItem key={system.id} value={system.id}>
                    {system.system_name}
                    {system.description && (
                      <span>({system.description})</span>
                    )}
                  </ComboboxItem>
                ))}
            </ComboboxList>
          </ComboboxContent>
        </>
      )}
      {isDisabled && value && (
        <p className="text-muted-foreground">
          Education System: {systems.find((s) => s.id === value)?.system_name ?? value}
        </p>
      )}
    </Combobox>
  );
}
