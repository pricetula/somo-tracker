/**
 * Non-blocking top banner for session recovery.
 *
 * Displayed on mount if an unfinished import session is found in IndexedDB.
 * Offers Resume or Discard & Start New actions.
 */

"use client";

import * as React from "react";
import type { ImportSession } from "../types";

interface SessionRecoveryBannerProps {
    session: ImportSession;
    onResume: () => void;
    onDiscard: () => void;
}

export function SessionRecoveryBanner({
    session,
    onResume,
    onDiscard,
}: SessionRecoveryBannerProps) {
    const timestamp = new Date(session.createdAt).toLocaleString();

    return (
        <div className="bg-muted/30 flex items-center justify-between px-4 py-3">
            <p className="text-foreground text-sm">
                You have an unfinished import session from{" "}
                <span className="font-medium">{timestamp}</span>. Resume or start fresh?
            </p>
            <div className="flex items-center gap-2">
                <button
                    onClick={onResume}
                    className="bg-primary text-primary-foreground hover:bg-primary/90 rounded-md px-3 py-1.5 text-xs font-medium"
                >
                    Resume Session
                </button>
                <button
                    onClick={onDiscard}
                    className="text-muted-foreground hover:text-foreground rounded-md px-3 py-1.5 text-xs font-medium"
                >
                    Discard & Start New
                </button>
            </div>
        </div>
    );
}
