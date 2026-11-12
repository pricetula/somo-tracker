/**
 * EducationSystemCombobox - Searchable education system selector for onboarding
 * Self-contained module: API + component
 * Depends on a selected countryId; disabled until country is chosen.
 */

'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { EducationSystem } from '../types';
import { listEducationSystems } from './education-system-combobox.api';

interface EducationSystemComboboxProps {
  value: string | null; // education system ID or null
  countryId: string | null; // required to fetch systems
  onChange: (educationSystemId: string | null) => void;
  disabled?: boolean;
}

export function EducationSystemCombobox({
  value,
  countryId,
  onChange,
  disabled = false,
}: EducationSystemComboboxProps) {
  const [systems, setSystems] = useState<EducationSystem[]>([]);
  const [filtered, setFiltered] = useState<EducationSystem[]>([]);
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [highlighted, setHighlighted] = useState(-1);

  const inputRef = useRef<HTMLInputElement>(null);
  const listboxRef = useRef<HTMLUListElement>(null);

  // Fetch education systems when countryId changes
  useEffect(() => {
    if (!countryId) {
      setSystems([]);
      setFiltered([]);
      setOpen(false);
      setHighlighted(-1);
      setSearch('');
      onChange(null);
      return;
    }

    listEducationSystems(countryId)
      .then((data) => {
        setSystems(data);
        setFiltered(data);
      })
      .catch(() => {
        // Error handled by toast in parent or silently
      });
  }, [countryId, onChange]);

  // Filter systems as user types
  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      setSearch(val);
      setOpen(true);
      setHighlighted(-1);
      const filteredList = systems.filter((s) =>
        s.system_name?.toLowerCase().includes(val.toLowerCase())
      );
      setFiltered(filteredList);
    },
    [systems]
  );

  // Select a system
  const selectSystem = useCallback(
    (system: EducationSystem) => {
      setSearch(system.system_name ?? '');
      setOpen(false);
      setHighlighted(-1);
      if (inputRef.current) inputRef.current.blur();
      onChange(system.id ?? null);
    },
    [onChange]
  );

  // Clear selection
  const clearSelection = useCallback(() => {
    setSearch('');
    setOpen(false);
    setHighlighted(-1);
    onChange(null);
  }, [onChange]);

  // Keyboard navigation
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (!open) {
        if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
          e.preventDefault();
          setOpen(true);
          setHighlighted(0);
        }
        return;
      }

      switch (e.key) {
        case 'Escape':
          e.preventDefault();
          setOpen(false);
          setHighlighted(-1);
          break;
        case 'ArrowDown':
          e.preventDefault();
          setHighlighted((prev) => (prev + 1) % filtered.length);
          break;
        case 'ArrowUp':
          e.preventDefault();
          setHighlighted((prev) => (prev - 1 + filtered.length) % filtered.length);
          break;
        case 'Enter':
          e.preventDefault();
          if (highlighted >= 0 && highlighted < filtered.length) {
            selectSystem(filtered[highlighted]);
          }
          break;
      }
    },
    [open, filtered, highlighted, selectSystem]
  );

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (inputRef.current && !inputRef.current.contains(e.target as Node)) {
        if (listboxRef.current && !listboxRef.current.contains(e.target as Node)) {
          setOpen(false);
          setHighlighted(-1);
        }
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Scroll highlighted into view
  useEffect(() => {
    if (!open || highlighted < 0 || !listboxRef.current) return;
    const activeOption = listboxRef.current.querySelector(`[data-index="${highlighted}"]`);
    if (!activeOption) return;
    const listbox = listboxRef.current;
    const optionRect = activeOption.getBoundingClientRect();
    const listboxRect = listbox.getBoundingClientRect();
    if (optionRect.top < listboxRect.top || optionRect.bottom > listboxRect.bottom) {
      activeOption.scrollIntoView({ block: 'nearest' });
    }
  }, [open, highlighted]);

  const isDisabled = disabled || !countryId;

  return (
    <div className="space-y-2">
      <Label htmlFor="education-system-combobox">Education System</Label>
      {!countryId ? (
        <p className="text-xs text-muted-foreground">
          Select a country first to view education systems.
        </p>
      ) : (
        <div
          role="combobox"
          aria-expanded={open}
          aria-haspopup="listbox"
          aria-controls="education-system-listbox"
          aria-activedescendant={highlighted >= 0 ? `education-system-option-${highlighted}` : undefined}
          className="relative"
        >
          <Input
            id="education-system-combobox"
            ref={inputRef}
            type="text"
            value={search}
            onChange={handleSearchChange}
            onKeyDown={handleKeyDown}
            onFocus={() => setOpen(true)}
            disabled={isDisabled}
            placeholder="Select an education system..."
            autoComplete="off"
          />
          {open && (
            <ul
              id="education-system-listbox"
              ref={listboxRef}
              role="listbox"
              aria-label="Education Systems"
              className="absolute z-10 mt-1 w-full border border-input bg-background rounded-md shadow-lg overflow-hidden max-h-60 overflow-y-auto"
            >
              {filtered.length === 0 ? (
                <li role="option" className="px-3 py-2 text-sm text-muted-foreground">
                  No education systems found
                </li>
              ) : (
                filtered.map((system, index) => {
                  const isSelected = system.id === value;
                  const isHighlighted = index === highlighted;
                  return (
                    <li
                      key={system.id}
                      role="option"
                      id={`education-system-option-${index}`}
                      data-index={index}
                      aria-selected={isSelected}
                      className={`cursor-pointer px-3 py-2 text-sm ${
                        isHighlighted
                          ? 'bg-primary text-primary-foreground'
                          : isSelected
                            ? 'bg-muted text-muted-foreground font-medium'
                            : 'text-foreground hover:bg-accent hover:text-accent-foreground'
                      }`}
                      onMouseMove={() => setHighlighted(index)}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        selectSystem(system);
                      }}
                    >
                      {system.system_name}
                      {system.description && <span className="text-xs opacity-50 ml-2">({system.description})</span>}
                      {isSelected && ' <span class="text-xs opacity-50">(selected)</span>'}
                    </li>
                  );
                })
              )}
            </ul>
          )}
          {value && !isDisabled && (
            <button
              type="button"
              onClick={clearSelection}
              className="absolute right-2 top-1/2 -translate-y-1/2 opacity-50 hover:opacity-100"
              aria-label="Clear selection"
            >
              ✕
            </button>
          )}
        </div>
      )}
      {isDisabled && value && (
        <p className="text-xs text-muted-foreground">
          Education System: {systems.find((s) => s.id === value)?.system_name ?? value}
        </p>
      )}
    </div>
  );
}
