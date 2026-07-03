/**
 * Import Stage Switcher — renders the current stage component based on stage state.
 *
 * Reads stage from local state; each stage component calls onStageChange to
 * advance. On mount, always starts in MAPPING — the Mapping component detects
 * persisted state and auto-advances if needed.
 */

"use client";

import * as React from "react";

import { ImportStageMapping } from "./import-stage-mapping";
import { ImportStagePreview } from "./import-stage-preview";
import { ImportStageReady } from "./import-stage-ready";
import { ImportStageSubmitting } from "./import-stage-submitting";

// ─── Props ─────────────────────────────────────────────────────────────────

interface ImportStageSwitcherProps {
    onClose: () => void;
}

// ─── Stage type ────────────────────────────────────────────────────────────

type Stage = "MAPPING" | "PREVIEW" | "READY" | "SUBMITTING";

// ─── Component ─────────────────────────────────────────────────────────────

export function ImportStageSwitcher({ onClose }: ImportStageSwitcherProps) {
    // Initialize to MAPPING directly — no useEffect needed for initial state.
    // The Mapping component checks persisted state and auto-advances if needed.
    const [stage, setStageLocal] = React.useState<Stage>("MAPPING");

    const handleStageChange = React.useCallback((nextStage: Stage) => {
        setStageLocal(nextStage);
    }, []);

    switch (stage) {
        case "MAPPING":
            return <ImportStageMapping onStageChange={handleStageChange} onClose={onClose} />;
        case "PREVIEW":
            return <ImportStagePreview onStageChange={handleStageChange} onClose={onClose} />;
        case "READY":
            return <ImportStageReady onStageChange={handleStageChange} onClose={onClose} />;
        case "SUBMITTING":
            return <ImportStageSubmitting onStageChange={handleStageChange} onClose={onClose} />;
        default:
            return <ImportStageMapping onStageChange={handleStageChange} onClose={onClose} />;
    }
}
