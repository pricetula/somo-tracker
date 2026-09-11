"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import { UploadFile } from "./upload-file";
import { FieldMapper, FieldDef, ExtractedRow, MappingResult, MappedRow } from "./field-mapper";

interface UploadProps {
    onCancel: () => void;
    fieldDef: FieldDef[];
    onMappedList: (rows: MappedRow[]) => void;
}

export function Upload({ onCancel, fieldDef, onMappedList }: UploadProps) {
    const [extractedRows, setExtractedRows] = React.useState<ExtractedRow[]>([]);

    const handleMappingChange = (result: MappingResult) => {
        onMappedList?.(result.rows);
    };

    const handleUploaded = (raw: Record<string, object>[]) => {
        const withIds: ExtractedRow[] = raw.map((r, i) => ({
            __id:
                typeof crypto !== "undefined" && crypto.randomUUID
                    ? crypto.randomUUID()
                    : `row-${i}`,
            ...r,
        })) as ExtractedRow[];
        setExtractedRows(withIds);
    };

    return (
        <div className="flex h-full flex-col">
            <div className="flex items-center justify-between">
                <h2 className="text-foreground font-semibold tracking-tight">Import admins</h2>
                <Button size="icon" variant="outline" onClick={onCancel} aria-label="Cancel">
                    <X className="h-4 w-4" />
                </Button>
            </div>

            <div className="mt-6 flex-1 space-y-6 overflow-auto">
                {extractedRows.length === 0 && <UploadFile onUploaded={handleUploaded} />}

                {extractedRows.length > 0 && (
                    <FieldMapper
                        extractedRows={extractedRows}
                        desiredFields={fieldDef}
                        onMappingChange={handleMappingChange}
                    />
                )}
            </div>
        </div>
    );
}
