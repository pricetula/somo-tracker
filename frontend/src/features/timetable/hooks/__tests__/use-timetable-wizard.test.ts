import { renderHook, act } from "@testing-library/react";
import { useTimetableWizard } from "../use-timetable-wizard";

describe("useTimetableWizard", () => {
    it("initializes with one valid slot", () => {
        const { result } = renderHook(() => useTimetableWizard());
        expect(result.current.slots).toHaveLength(1);
        expect(result.current.slotsValid).toBe(true);
        expect(result.current.hasGapsOrOverlap).toBe(false);
    });

    it("validates start < end for each slot", () => {
        const { result } = renderHook(() => useTimetableWizard());
        act(() => {
            result.current.updateSlot(result.current.slots[0].id, {
                start_time: "10:00",
                end_time: "09:00",
            });
        });
        expect(result.current.slotsValid).toBe(false);
    });

    it("detects a gap between consecutive slots", () => {
        const { result } = renderHook(() => useTimetableWizard());
        act(() => {
            result.current.addSlot();
        });
        // slots[0] end 09:00, slots[1] start 09:00 → no gap
        expect(result.current.hasGapsOrOverlap).toBe(false);

        act(() => {
            const secondId = result.current.slots[1].id;
            result.current.updateSlot(secondId, { start_time: "09:30" });
        });
        expect(result.current.hasGapsOrOverlap).toBe(true);
    });

    it("cascades end time to next slot start", () => {
        const { result } = renderHook(() => useTimetableWizard());
        act(() => {
            result.current.addSlot();
        });
        const firstId = result.current.slots[0].id;
        act(() => {
            result.current.updateSlot(firstId, { end_time: "09:30" });
        });
        expect(result.current.slots[1].start_time).toBe("09:30");
        expect(result.current.hasGapsOrOverlap).toBe(false);
    });

    it("prevents deleting last slot", () => {
        const { result } = renderHook(() => useTimetableWizard());
        const id = result.current.slots[0].id;
        act(() => {
            result.current.deleteSlot(id);
        });
        expect(result.current.slots).toHaveLength(1);
    });
});
