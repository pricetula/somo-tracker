/**
 * Ingestion vector selector — toggle between Manual Entry and File Upload.
 */

"use client";

import { FileSpreadsheet, Keyboard } from "lucide-react";

interface IngestionSelectorProps {
    onSelect: (mode: "manual" | "csv") => void;
}

export function IngestionSelector({ onSelect }: IngestionSelectorProps) {
    return (
        <div className="space-y-4">
            <h2 className="text-lg font-semibold">Import Students</h2>
            <p className="text-muted-foreground text-sm">
                Choose how you would like to add students to the system.
            </p>

            <div className="grid grid-cols-2 gap-4 pt-2">
                <button
                    onClick={() => onSelect("manual")}
                    className="bg-muted/30 hover:bg-muted/50 flex flex-col items-center gap-3 rounded-md p-6 text-left transition-colors"
                >
                    <Keyboard className="text-primary size-8" />
                    <div className="text-center">
                        <p className="text-sm font-medium">Manual Entry</p>
                        <p className="text-muted-foreground mt-1 text-xs">
                            Add students one by one using an inline form grid
                        </p>
                    </div>
                </button>

                <button
                    onClick={() => onSelect("csv")}
                    className="bg-muted/30 hover:bg-muted/50 flex flex-col items-center gap-3 rounded-md p-6 text-left transition-colors"
                >
                    <FileSpreadsheet className="text-primary size-8" />
                    <div className="text-center">
                        <p className="text-sm font-medium">Upload File</p>
                        <p className="text-muted-foreground mt-1 text-xs">
                            Import from CSV or Excel with column mapping wizard
                        </p>
                    </div>
                </button>
            </div>

            <aside className="bg-muted/20 text-muted-foreground rounded-md px-3 py-2 text-xs">
                Max file size: 10MB &middot; Max rows: 5,000
            </aside>
        </div>
    );
}
