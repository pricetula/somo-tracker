"use client";

import { useCallback, useMemo } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { SortingState } from "@tanstack/react-table";

interface UrlState {
  search: string;
  sorting: SortingState;
  filters: Record<string, unknown>;
}

export function readFromUrl(
  searchParams: URLSearchParams,
  initialFilters: Record<string, unknown>,
  initialSorting: SortingState,
): UrlState {
  const search = searchParams.get("search") ?? "";

  const sort = searchParams.get("sort");
  const order = searchParams.get("order");
  const sorting: SortingState = sort
    ? [{ id: sort, desc: order === "desc" }]
    : initialSorting;

  const filters: Record<string, unknown> = { ...initialFilters };
  for (const [key, value] of searchParams.entries()) {
    if (
      key === "search" ||
      key === "sort" ||
      key === "order" ||
      !(key in initialFilters)
    ) {
      continue;
    }
    const all = searchParams.getAll(key);
    filters[key] = all.length > 1 ? all : all.length === 1 ? value : "";
  }

  return { search, sorting, filters };
}

export function useDataTableUrlState(
  initialFilters: Record<string, unknown>,
  initialSorting: SortingState = [],
) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  // State is derived from the URL on every render, so browser back/forward
  // navigation restores table state without any effect-driven setState.
  const state = useMemo(
    () => readFromUrl(searchParams, initialFilters, initialSorting),
    [searchParams, initialFilters, initialSorting],
  );

  const commit = useCallback(
    (next: UrlState, method: "push" | "replace" = "push") => {
      const params = new URLSearchParams();
      if (next.search) {
        params.set("search", next.search);
      }
      if (next.sorting.length) {
        params.set("sort", next.sorting[0].id);
        params.set("order", next.sorting[0].desc ? "desc" : "asc");
      }
      for (const [id, value] of Object.entries(next.filters)) {
        if (value === undefined || value === null || value === "") {
          continue;
        }
        if (Array.isArray(value) && value.length === 0) {
          continue;
        }
        const values = Array.isArray(value) ? value : [String(value)];
        for (const v of values) {
          params.append(id, String(v));
        }
      }
      const url = params.toString()
        ? `${pathname}?${params.toString()}`
        : pathname;
      const options = { scroll: false };
      if (method === "replace") {
        router.replace(url, options);
      } else {
        router.push(url, options);
      }
    },
    [pathname, router],
  );

  const setSearch = useCallback(
    (value: string) => {
      // Transient search updates replace history to avoid excessive entries.
      commit({ ...state, search: value }, "replace");
    },
    [state, commit],
  );

  const setSorting = useCallback(
    (next: SortingState) => {
      commit({ ...state, sorting: next });
    },
    [state, commit],
  );

  const setFilters = useCallback(
    (next: Record<string, unknown>) => {
      commit({ ...state, filters: next });
    },
    [state, commit],
  );

  return {
    search: state.search,
    sorting: state.sorting,
    filters: state.filters,
    setSearch,
    setSorting,
    setFilters,
  };
}
