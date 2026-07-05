import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { type ListApiFn, type NormalizedListResult } from "./types";
import { defaultNormalize } from "./utils";

export interface UseInfiniteListQueryOptions<TItem, TParams extends object, TResult> {
    /** Base query key, e.g. ["classes"]. `params`, `search`, and `filters` are appended automatically. */
    queryKey: readonly unknown[];
    /** Any generated `list*` function, e.g. `listClasses`. */
    queryFn: ListApiFn<TResult, TParams>;
    /** Resource-specific filters (excluding page/limit/search). */
    params: TParams;
    /** Search term (debounced). When it changes, the query resets to page 1. */
    search?: string;
    /** Active filter values, keyed by filter group id. When it changes, the query resets to page 1. */
    filters?: Record<string, string[]>;
    /** Rows fetched per page. Defaults to 50. */
    limit?: number;
    /** Only needed if the generated result's shape differs from NormalizedListResult. */
    normalize?: (result: TResult) => NormalizedListResult<TItem>;
    enabled?: boolean;
}

/**
 * Infinite, page-accumulating query built on TanStack Query v5's
 * useInfiniteQuery. Pairs with a row virtualizer: as the user scrolls near
 * the end of the rendered rows, the table calls fetchNextPage(), and the
 * new page's rows are appended to the flattened `rows` array below.
 *
 * The `search` and `filters` values are included in the query key so that
 * changing either one causes a full refetch from page 1.
 */
export function useInfiniteListQuery<TItem, TParams extends object, TResult>({
    queryKey,
    queryFn,
    params,
    search,
    filters,
    limit = 50,
    normalize = defaultNormalize<TItem, TResult>,
    enabled = true,
}: UseInfiniteListQueryOptions<TItem, TParams, TResult>) {
    const query = useInfiniteQuery({
        queryKey: [...queryKey, params, search ?? "", filters ?? {}, limit],
        queryFn: ({ pageParam }) =>
            queryFn({
                ...params,
                ...(search ? { search } : {}),
                ...(filters && Object.keys(filters).length > 0 ? { filters } : {}),
                page: pageParam,
                limit,
            } as TParams & { page?: number; limit?: number }),
        initialPageParam: 1,
        getNextPageParam: (lastPage, allPages) => {
            const normalized = normalize(lastPage);

            if (normalized.hasMore !== undefined) {
                return normalized.hasMore ? allPages.length + 1 : undefined;
            }

            if (normalized.total !== undefined) {
                const loaded = allPages.reduce(
                    (sum, page) => sum + normalize(page).items.length,
                    0
                );
                return loaded < normalized.total ? allPages.length + 1 : undefined;
            }

            // No total/hasMore to go on — assume a short page means the end.
            return normalized.items.length === limit ? allPages.length + 1 : undefined;
        },
        placeholderData: keepPreviousData,
        enabled,
    });

    const rows = useMemo(
        () => query.data?.pages.flatMap((page) => normalize(page).items) ?? [],
        [query.data, normalize]
    );

    const total = useMemo(
        () => (query.data ? normalize(query.data.pages[0]).total : undefined),
        [query.data, normalize]
    );

    return { ...query, rows, total };
}
