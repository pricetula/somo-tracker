// types.test.ts
import { defaultNormalize } from "../utils";

describe("defaultNormalize", () => {
    it("returns input unchanged (identity passthrough)", () => {
        const input = { items: [1, 2, 3], total: 100, page: 1, limit: 50 };
        const result = defaultNormalize(input);
        expect(result).toBe(input);
        expect(result.items).toEqual([1, 2, 3]);
        expect(result.total).toBe(100);
    });

    it("works with minimal object", () => {
        const input = { items: [] };
        const result = defaultNormalize(input);
        expect(result).toBe(input);
        expect(result.items).toEqual([]);
    });
});
