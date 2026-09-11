"use client";

import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Upload } from "./upload";
import { ManualImport } from "./manual-import";

/**
 * ImportOrchestrator — central state machine for the data import flow.
 *
 * Manages the full lifecycle: SELECT_MODE → UPLOAD → MAPPING → VALIDATING →
 * PREVIEW_HAPPY / PREVIEW_ERRORS → RESOLVING_ERRORS → REVALIDATING →
 * SUBMITTING → SUBMIT_SUCCESS / SUBMIT_ERROR.
 *
 * State transitions follow the spec exactly.
 */

export function ImportOrchestrator() {
    const [importType, setImportType] = React.useState("");

    return (
        <div className="relative flex h-100 gap-4 overflow-hidden">
            <AnimatePresence mode="wait">
                {importType === "manual" ? (
                    <motion.div
                        key="manual"
                        initial={{ opacity: 0, y: 12, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -12, scale: 0.98 }}
                        transition={{ duration: 0.25, ease: "easeInOut" }}
                        className="h-full w-full"
                    >
                        <ManualImport onCancel={() => setImportType("")} />
                    </motion.div>
                ) : importType === "upload" ? (
                    <motion.div
                        key="upload"
                        initial={{ opacity: 0, y: 12, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -12, scale: 0.98 }}
                        transition={{ duration: 0.25, ease: "easeInOut" }}
                        className="h-full w-full"
                    >
                        <Upload onCancel={() => setImportType("")} />
                    </motion.div>
                ) : (
                    <motion.div
                        key="select"
                        initial={{ opacity: 0, y: 12, scale: 0.98 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: -12, scale: 0.98 }}
                        transition={{ duration: 0.25, ease: "easeInOut" }}
                        className="flex h-full w-full items-center justify-center gap-4"
                    >
                        <Button onClick={() => setImportType("manual")}>Manual Import</Button>
                        <Button onClick={() => setImportType("upload")}>Upload</Button>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
}
