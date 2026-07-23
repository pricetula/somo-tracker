import { useEffect, useState } from "react";

/**
 * Returns a debounced copy of `value` that only updates after `delay` ms
 * of no changes. Used to avoid firing `onSearch` (and therefore a network
 * request) on every keystroke.
 */
export function useDebouncedValue<T>(value: T, delay = 350): T {
    const [debounced, setDebounced] = useState(value);

    useEffect(() => {
        const timeout = setTimeout(() => setDebounced(value), delay);
        return () => clearTimeout(timeout);
    }, [value, delay]);

    return debounced;
}
