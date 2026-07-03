import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import type { ListApiFn, NormalizedListResult } from "./types";
import { defaultNormalize } from "./utils";

export interface UseInfiniteListQueryOptions<TItem, TParams extends object, TResult> {
    /** Base query key, e.g. ["classes"]. `params` is appended automatically. */
    queryKey: readonly unknown[];
    /** Any generated `list*` function, e.g. `listClasses`. */
    queryFn: ListApiFn<TResult, TParams>;
    /** Resource-specific filters (excluding page/limit). */
    params: TParams;
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
 */
export function useInfiniteListQuery<TItem, TParams extends object, TResult>({
    queryKey,
    queryFn,
    params,
    limit = 50,
    normalize = defaultNormalize<TItem, TResult>,
    enabled = true,
}: UseInfiniteListQueryOptions<TItem, TParams, TResult>) {
    const query = useInfiniteQuery({
        queryKey: [...queryKey, params, limit],
        queryFn: ({ pageParam }) => queryFn({ ...params, page: pageParam, limit }),
        initialPageParam: 1,
        getNextPageParam: (lastPage, allPages) => {
            const normalized = normalize(lastPage);

            if (normalized.hasMore !== undefined) {
                return normalized?.hasMore && allPages?.length ? allPages.length + 1 : undefined;
            }

            if (normalized?.total !== undefined) {
                const loaded = allPages?.reduce?.(
                    (sum, page) => sum + normalize(page).items.length,
                    0
                );
                return loaded < normalized?.total && allPages?.length
                    ? allPages.length + 1
                    : undefined;
            }

            // No total/hasMore to go on — assume a short page means the end.
            return normalized?.items.length === limit && allPages?.length
                ? allPages.length + 1
                : undefined;
        },
        placeholderData: keepPreviousData,
        enabled,
    });

    const rows = useMemo(
        () => query?.data?.pages?.flatMap?.((page) => normalize(page).items) ?? [],
        [query.data, normalize]
    );

    const total = useMemo(
        () => (query?.data ? normalize(query?.data?.pages?.[0])?.total : undefined),
        [query.data, normalize]
    );

    return { ...query, rows, total };
}
