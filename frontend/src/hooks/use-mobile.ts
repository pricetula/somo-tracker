import * as React from "react";

const MOBILE_BREAKPOINT = 768;

function getIsMobile(): boolean {
    return window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`).matches;
}

function subscribe(onStoreChange: () => void): () => void {
    const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
    mql.addEventListener("change", onStoreChange);
    return () => mql.removeEventListener("change", onStoreChange);
}

export function useIsMobile(): boolean {
    return React.useSyncExternalStore(
        subscribe,
        getIsMobile,
        () => false // getServerSnapshot — always false on the server
    );
}
