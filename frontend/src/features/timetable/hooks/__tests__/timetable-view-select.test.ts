import { describe, it, expect } from "vitest";
import { timetableViewSelect } from "../use-timetable";
import type { TimeBlock, Allocation } from "@/lib/api/timetable";

function makeBlock(overrides: Partial<TimeBlock> = {}): TimeBlock {
    return {
        id: `block-${Math.random().toString(36).slice(2)}`,
        track_id: "track-1",
        day_of_week: 1,
        period_name: "Period 1",
        start_time: "08:00",
        end_time: "09:00",
        is_break: false,
        order: 1,
        ...overrides,
    };
}

function makeAllocation(overrides: Partial<Allocation> = {}): Allocation {
    return {
        id: `alloc-${Math.random().toString(36).slice(2)}`,
        tenant_id: "t-1",
        school_id: "s-1",
        academic_year_id: "ay-1",
        block_id: overrides.block_id ?? "block-1",
        class_id: "c-1",
        learning_area_id: "la-1",
        teacher_id: "tch-1",
        room_identifier: null,
        class_name: "Class A",
        learning_area_name: "Math",
        learning_area_code: "MATH",
        teacher_name: "Ms. Smith",
        ...overrides,
    };
}

describe("timetableViewSelect", () => {
    it("returns data when blocks and allocations are present", () => {
        const block = makeBlock({
            id: "b1",
            day_of_week: 2,
            period_name: "P2",
            start_time: "09:00",
            end_time: "10:00",
            order: 2,
        });
        const allocation = makeAllocation({ block_id: "b1" });

        const result = timetableViewSelect({ blocks: [block], allocations: [allocation] });

        expect(result.days).toHaveLength(1);
        expect(result.days[0].day_of_week).toBe(2);
        expect(result.rows).toHaveLength(1);
        expect(result.rows[0].period_name).toBe("P2");
        expect(result.rows[0].allocationByDay[2]).toBe(allocation);
    });

    it("returns empty days and rows when no data is provided", () => {
        const result = timetableViewSelect({ blocks: [], allocations: [] });
        expect(result.days).toEqual([]);
        expect(result.rows).toEqual([]);
    });
});
