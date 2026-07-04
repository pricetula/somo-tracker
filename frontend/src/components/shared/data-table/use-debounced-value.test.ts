// use-debounced-value.test.ts
import { renderHook, act } from "@testing-library/react";
import { vi } from "vitest";
import { useDebouncedValue } from "@/components/shared/data-table/use-debounced-value";

describe("useDebouncedValue", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it("returns the initial value immediately", () => {
        const { result } = renderHook(() => useDebouncedValue("initial", 100));
        expect(result.current).toBe("initial");
    });

    it("does not update before delay ms", () => {
        const { result } = renderHook(() => useDebouncedValue("initial", 100));
        act(() => {
            result.current = "changed";
        });
        expect(result.current).toBe("initial");
        vi.advanceTimersByTime(50);
        expect(result.current).toBe("initial");
    });

    it("updates to new value after delay", () => {
        const { result } = renderHook(() => useDebouncedValue("initial", 100));
        act(() => {
            result.current = "changed";
        });
        vi.advanceTimersByTime(100);
        expect(result.current).toBe("changed");
    });

    it("only returns last value when changed rapidly", () => {
        const { result } = renderHook(() => useDebouncedValue("initial", 100));
        act(() => {
            result.current = "first";
            result.current = "second";
            result.current = "third";
        });
        vi.advanceTimersByTime(50);
        expect(result.current).toBe("initial"); // still initial
        vi.advanceTimersByTime(100);
        expect(result.current).toBe("third");
    });

    it("cleans up timeout on unmount", () => {
        const clearTimeoutSpy = vi.spyOn(window, "clearTimeout");
        const { unmount } = renderHook(() => useDebouncedValue("value", 100));
        unmount();
        expect(clearTimeoutSpy).toHaveBeenCalled();
    });

    it("works with custom delay", () => {
        const { result } = renderHook(() => useDebouncedValue("initial", 50));
        act(() => {
            result.current = "changed";
        });
        vi.advanceTimersByTime(30);
        expect(result.current).toBe("initial");
        vi.advanceTimersByTime(30);
        expect(result.current).toBe("initial");
        vi.advanceTimersByTime(20);
        expect(result.current).toBe("changed");
    });
});
