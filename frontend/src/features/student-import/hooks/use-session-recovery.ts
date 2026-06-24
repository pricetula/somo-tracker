/**
 * Session recovery hook.
 *
 * On mount, checks IndexedDB for an unfinished import session.
 * Returns state + actions for resume/discard flow.
 */

"use client";

import * as React from "react";
import { hasStoredSession, loadSession, clearSession } from "../lib/indexeddb";
import type { ImportSession } from "../types";

export type RecoveryAction = "loading" | "prompt" | "clear";

export interface SessionRecoveryState {
    action: RecoveryAction;
    session: ImportSession | null;
    resume: () => void;
    discard: () => void;
}

export function useSessionRecovery(): SessionRecoveryState {
    const [action, setAction] = React.useState<RecoveryAction>("loading");
    const [session, setSession] = React.useState<ImportSession | null>(null);

    React.useEffect(() => {
        let cancelled = false;

        async function check() {
            const exists = await hasStoredSession();
            if (cancelled) return;

            if (exists) {
                const s = await loadSession();
                if (cancelled) return;
                if (s) {
                    setSession(s);
                    setAction("prompt");
                } else {
                    setAction("clear");
                }
            } else {
                setAction("clear");
            }
        }

        check().catch(() => {
            if (!cancelled) setAction("clear");
        });

        return () => {
            cancelled = true;
        };
    }, []);

    const resume = React.useCallback(() => {
        setAction("clear"); // Caller reconstructs state from IndexedDB
    }, []);

    const discard = React.useCallback(async () => {
        await clearSession();
        setSession(null);
        setAction("clear");
    }, []);

    return { action, session, resume, discard };
}
