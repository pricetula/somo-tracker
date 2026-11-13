'use client';

import { useEffect, useRef, useState, useCallback } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { Country } from '../types';
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
  const [filtered, setFiltered] = useState<Country[]>([]);
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [highlighted, setHighlighted] = useState(-1);

  const inputRef = useRef<HTMLInputElement>(null);
  const listboxRef = useRef<HTMLUListElement>(null);

  // Filter countries as user types
  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = e.target.value;
      setSearch(val);
      setOpen(true);
      setHighlighted(-1);
      const filteredList = countries.filter((c) =>
        c.country_name?.toLowerCase().includes(val.toLowerCase())
      );
      setFiltered(filteredList);
    },
    [countries]
  );

  // Select a country
  const selectCountry = useCallback(
    (country: Country) => {
      setSearch(country.country_name ?? '');
      setOpen(false);
      setHighlighted(-1);
      if (inputRef.current) inputRef.current.blur();
      onChange(country.id ?? null);
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
            selectCountry(filtered[highlighted]);
          }
          break;
      }
    },
    [open, filtered, highlighted, selectCountry]
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

  // Sync highlighted with visible listbox scroll
  useEffect(() => {
    if (!open || highlighted < 0 || !listboxRef.current) return;
    const activeOption = listboxRef.current.querySelector(
      `[data-index="${highlighted}"]`
    );
    if (!activeOption) return;
    const listbox = listboxRef.current;
    const optionRect = activeOption.getBoundingClientRect();
    const listboxRect = listbox.getBoundingClientRect();
    if (optionRect.top < listboxRect.top || optionRect.bottom > listboxRect.bottom) {
      activeOption.scrollIntoView({ block: 'nearest' });
    }
  }, [open, highlighted]);

  return (
    <div className="space-y-2">
      <Label htmlFor="country-combobox">Country</Label>
      <div
        role="combobox"
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-controls="country-listbox"
        aria-activedescendant={
          highlighted >= 0 ? `country-option-${highlighted}` : undefined
        }
        className="relative"
      >
        <Input
          id="country-combobox"
          ref={inputRef}
          type="text"
          value={search}
          onChange={handleSearchChange}
          onKeyDown={handleKeyDown}
          onFocus={() => setOpen(true)}
          disabled={disabled}
          placeholder="Select a country..."
          autoComplete="off"
        />
        {open && (
          <ul
            id="country-listbox"
            ref={listboxRef}
            role="listbox"
            aria-label="Countries"
            className="absolute z-10 mt-1 w-full max-h-60 overflow-y-auto"
          >
            {isLoading ? (
              <li role="option" className="px-3 py-2">
                Loading...
              </li>
            ) : filtered.length === 0 ? (
              <li role="option" className="px-3 py-2">
                No countries found
              </li>
            ) : (
              filtered.map((country, index) => {
                const isSelected = country.id === value;
                const isHighlighted = index === highlighted;
                return (
                  <li
                    key={country.id}
                    role="option"
                    id={`country-option-${index}`}
                    data-index={index}
                    aria-selected={isSelected}
                    className={`cursor-pointer px-3 py-2 ${isHighlighted ? 'bg-primary text-primary-foreground' : isSelected ? 'bg-muted text-muted-foreground font-medium' : 'text-foreground hover:bg-accent hover:text-accent-foreground'}`}
                    onMouseMove={() => setHighlighted(index)}
                    onMouseDown={(e) => {
                      e.preventDefault();
                      selectCountry(country);
                    }}
                  >
                    {country.country_name} ({country.country_code})
                    {isSelected && ' <span class="text-xs opacity-50">(selected)</span>'}
                  </li>
                );
              })
            )}
          </ul>
        )}
        {value && !disabled && (
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
      {disabled && value && (
        <p className="text-xs text-muted-foreground">
          Country: {countries.find((c) => c.id === value)?.country_name ?? value}
        </p>
      )}
    </div>
  );
}
