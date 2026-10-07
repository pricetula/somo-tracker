// types.test.ts
import { defaultNormalize } from "../utils";

describe("defaultNormalize", () => {
    it("returns input unchanged (identity passthrough)", () => {
        const input = { items: [1, 2, 3], total: 100, page: 1, limit: 50 };
        const result = defaultNormalize(input);
        expect(result).toEqual(input);
        expect(result.items).toEqual([1, 2, 3]);
        expect(result.total).toBe(100);
    });

    it("works with minimal object", () => {
        const input = { items: [] };
        const result = defaultNormalize(input);
        expect(result).toEqual({
            items: [],
            total: undefined,
            page: undefined,
            limit: undefined,
            hasMore: undefined,
        });
        expect(result.items).toEqual([]);
    });

    it("coerces null items to empty array", () => {
        const input = { items: null, total: 0, page: 1, limit: 50 };
        const result = defaultNormalize(input);
        expect(result.items).toEqual([]);
        expect(result.total).toBe(0);
    });
});
