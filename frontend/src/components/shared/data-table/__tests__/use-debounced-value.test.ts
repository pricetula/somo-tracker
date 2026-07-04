import { act, renderHook } from "@testing-library/react";
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

    it("does not update before the delay has elapsed", () => {
        const { result, rerender } = renderHook(
            ({ value, delay }: { value: string; delay: number }) => useDebouncedValue(value, delay),
            { initialProps: { value: "initial", delay: 100 } }
        );

        rerender({ value: "changed", delay: 100 });
        expect(result.current).toBe("initial");

        act(() => {
            vi.advanceTimersByTime(50);
        });
        expect(result.current).toBe("initial");
    });

    it("updates to the new value after the delay", () => {
        const { result, rerender } = renderHook(
            ({ value, delay }: { value: string; delay: number }) => useDebouncedValue(value, delay),
            { initialProps: { value: "initial", delay: 100 } }
        );

        rerender({ value: "changed", delay: 100 });

        act(() => {
            vi.advanceTimersByTime(100);
        });
        expect(result.current).toBe("changed");
    });

    it("only returns the last value when changed rapidly", () => {
        const { result, rerender } = renderHook(
            ({ value, delay }: { value: string; delay: number }) => useDebouncedValue(value, delay),
            { initialProps: { value: "initial", delay: 100 } }
        );

        rerender({ value: "first", delay: 100 });
        rerender({ value: "second", delay: 100 });
        rerender({ value: "third", delay: 100 });

        act(() => {
            vi.advanceTimersByTime(50);
        });
        expect(result.current).toBe("initial");

        act(() => {
            vi.advanceTimersByTime(100);
        });
        expect(result.current).toBe("third");
    });

    it("clears the pending timeout on unmount", () => {
        const clearTimeoutSpy = vi.spyOn(globalThis, "clearTimeout");

        const { rerender, unmount } = renderHook(
            ({ value, delay }: { value: string; delay: number }) => useDebouncedValue(value, delay),
            { initialProps: { value: "initial", delay: 100 } }
        );

        rerender({ value: "changed", delay: 100 });
        unmount();

        expect(clearTimeoutSpy).toHaveBeenCalled();
        clearTimeoutSpy.mockRestore();
    });

    it("works with a custom delay", () => {
        const { result, rerender } = renderHook(
            ({ value, delay }: { value: string; delay: number }) => useDebouncedValue(value, delay),
            { initialProps: { value: "initial", delay: 50 } }
        );

        // Before the delay, value stays unchanged
        act(() => {
            vi.advanceTimersByTime(49);
        });
        expect(result.current).toBe("initial");

        // Rerender triggers a new debounce timer
        rerender({ value: "changed", delay: 50 });
        expect(result.current).toBe("initial");

        act(() => {
            vi.advanceTimersByTime(60);
        });
        expect(result.current).toBe("changed");
    });
});
