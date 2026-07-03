import { NormalizedListResult } from "./types";

/**
 * Default normalizer: assumes the result already matches
 * NormalizedListResult<TItem> (i.e. exposes `items`, `total`, `page`,
 * `limit` directly). Most Go/swagger-generated responses instead name the
 * items field after the resource (e.g. `{ students, total, page, limit }`)
 * — for those, use `normalizeListResponse()` below instead of this.
 */
export function defaultNormalize<TItem, TResult>(result: TResult): NormalizedListResult<TItem> {
    return result as unknown as NormalizedListResult<TItem>;
}

/**
 * Builds a `normalize` function for the common Go handler pattern:
 *
 *   type ListStudentsResponse struct {
 *       Students []Student `json:"students"`
 *       Total    int       `json:"total"`
 *       Page     int       `json:"page"`
 *       Limit    int       `json:"limit"`
 *   }
 *
 * i.e. any response shaped `{ [itemsKey]: TItem[], total, page, limit }`.
 * Usage:
 *
 *   normalize={normalizeListResponse("students")}
 *   normalize={normalizeListResponse("classes")}
 */
export function normalizeListResponse<TItem, TKey extends string>(itemsKey: TKey) {
    return (
        result: Record<TKey, TItem[]> & {
            total: number;
            page: number;
            limit: number;
        }
    ): NormalizedListResult<TItem> => ({
        items: result[itemsKey],
        total: result.total,
        page: result.page,
        limit: result.limit,
    });
}
