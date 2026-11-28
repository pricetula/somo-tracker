"use client";

import { useCallback, useRef, useState, useEffect } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
  ComboboxValue,
  ComboboxEmpty,
} from "@/components/ui/combobox";
import { Search } from "lucide-react";
import type { DataTableFilter } from "../types";

interface DataTableToolbarProps {
  search: string;
  setSearch: (value: string) => void;
  filters: DataTableFilter[];
  filterValues: Record<string, unknown>;
  onFilterChange: (values: Record<string, unknown>) => void;
  isLoading?: boolean;
  debounceMs?: number;
}

export function DataTableToolbar({
  search,
  setSearch,
  filters,
  filterValues,
  onFilterChange,
  isLoading = false,
  debounceMs = 300,
}: DataTableToolbarProps) {
  const [searchInput, setSearchInput] = useState(search);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Clear pending debounce timer on unmount to avoid state updates after
  // navigation / component removal.
  useEffect(() => {
    return () => {
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
    };
  }, []);

  const handleSearchChange = useCallback(
    (value: string) => {
      setSearchInput(value);
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
      searchTimerRef.current = setTimeout(() => {
        setSearch(value);
      }, debounceMs);
    },
    [setSearch, debounceMs],
  );

  const handleFilterChange = useCallback(
    (id: string, value: unknown) => {
      onFilterChange({ ...filterValues, [id]: value });
    },
    [filterValues, onFilterChange],
  );

  const hasFilters = filters.length > 0;

  return (
    <div className="flex flex-col sm:flex-row sm:items-end sm:gap-4 gap-3 border-b border-background pb-2">
      <div className="flex mb-1 bg-input/20 items-center gap-1 rounded-md border border-input px-2 focus-within:ring-2 focus-within:ring-ring/30">
        <Search className="size-4 text-muted-foreground" />
        <Input
          value={searchInput}
          onChange={(e) => handleSearchChange(e.target.value)}
          placeholder="Search…"
          disabled={isLoading}
          style={{ background: "transparent" }}
          className="border-0 outline-none ring-0 focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:outline-none h-6.5"
        />
      </div>

      {hasFilters && (
        <div className="mt-3 sm:mt-0 sm:w-[260px] space-y-2">
          {filters.map((filter) => {
            const value = filterValues[filter.id];
            const isMulti = filter.type === "multi-select";
            const current = isMulti
              ? Array.isArray(value)
                ? value
                : []
              : typeof value === "string"
                ? value
                : "";

            return (
              <div key={filter.id} className="space-y-1">
                <Label htmlFor={`filter-${filter.id}`}>{filter.label}</Label>
                {isMulti ? (
                  <Combobox
                    multiple
                    value={current as string[]}
                    onValueChange={(next) =>
                      handleFilterChange(filter.id, next ?? [])
                    }
                  >
                    <ComboboxInput
                      id={`filter-${filter.id}`}
                      placeholder={filter.placeholder ?? "Select…"}
                      showTrigger
                    />
                    <ComboboxContent>
                      <ComboboxEmpty>No options</ComboboxEmpty>
                      <ComboboxList>
                        {filter.options.map((option) => (
                          <ComboboxItem
                            key={option.value}
                            value={option.value}
                            disabled={option.disabled}
                          >
                            {option.label}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                ) : (
                  <Combobox
                    value={current}
                    onValueChange={(next) =>
                      handleFilterChange(filter.id, next ?? "")
                    }
                  >
                    <ComboboxTrigger>
                      <ComboboxValue>
                        {current || filter.placeholder}
                      </ComboboxValue>
                    </ComboboxTrigger>
                    <ComboboxContent>
                      <ComboboxEmpty>No options</ComboboxEmpty>
                      <ComboboxList>
                        <ComboboxItem value="">All</ComboboxItem>
                        {filter.options.map((option) => (
                          <ComboboxItem
                            key={option.value}
                            value={option.value}
                            disabled={option.disabled}
                          >
                            {option.label}
                          </ComboboxItem>
                        ))}
                      </ComboboxList>
                    </ComboboxContent>
                  </Combobox>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
