// types.test.ts
import { defaultNormalize, normalizeListResponse } from "../utils";

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

describe("normalizeListResponse", () => {
    it("extracts items from named key", () => {
        const normalize = normalizeListResponse("students");
        const input = {
            students: [
                { id: 1, name: "Alice" },
                { id: 2, name: "Bob" },
            ],
            total: 5,
            page: 1,
            limit: 20,
        };
        const result = normalize(input);
        expect(result.items).toEqual(input.students);
        expect(result.total).toBe(5);
        expect(result.page).toBe(1);
        expect(result.limit).toBe(20);
    });

    it("works with different key names", () => {
        // Test with "classes"
        const normalizeClasses = normalizeListResponse("classes");
        const classesInput = {
            classes: [{ id: 1, name: "Math" }],
            total: 10,
            page: 2,
            limit: 25,
        };
        const classesResult = normalizeClasses(classesInput);
        expect(classesResult.items).toEqual(classesInput.classes);
        expect(classesResult.total).toBe(10);

        // Test with "teachers"
        const normalizeTeachers = normalizeListResponse("teachers");
        const teachersInput = {
            teachers: [{ id: 1, name: "Mr. Smith" }],
            total: 30,
            page: 1,
            limit: 50,
        };
        const teachersResult = normalizeTeachers(teachersInput);
        expect(teachersResult.items).toEqual(teachersInput.teachers);
        expect(teachersResult.total).toBe(30);
    });

    it("extracts from various resource types", () => {
        const keys = ["students", "classes", "teachers", "parents", "assignments"] as const;
        keys.forEach((key, index) => {
            const normalize = normalizeListResponse(key);
            const items = [{ id: index, name: `${key} ${index}` }];
            const input = {
                [key]: items,
                total: 100,
                page: 1,
                limit: 50,
            };
            const result = normalize(
                input as Record<typeof key, typeof items> & {
                    total: number;
                    page: number;
                    limit: number;
                }
            );
            expect(result.items).toBe(items);
            expect(result.total).toBe(100);
        });
    });
});
