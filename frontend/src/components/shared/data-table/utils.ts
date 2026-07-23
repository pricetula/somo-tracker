import { NormalizedListResult } from "./types";

/**
 * Default normalizer: assumes the result already matches
 * NormalizedListResult<TItem> (i.e. exposes `items`, `total`, `page`,
 * `limit` directly).
 *
 * Defensively coerces `items` to an array — some API responses return
 * `null` instead of `[]` when the list is empty. Without this guard,
 * the DataTable's virtualizer would receive `[null]` as rows and crash
 * when rendering cells that access row properties.
 */
export function defaultNormalize<TItem, TResult>(result: TResult): NormalizedListResult<TItem> {
    const raw = result as unknown as NormalizedListResult<TItem>;
    return {
        items: raw.items ?? [],
        total: raw.total,
        page: raw.page,
        limit: raw.limit,
        hasMore: raw.hasMore,
    };
}
