"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import { X } from "lucide-react";
import { UploadFile } from "./upload-file";
import { FieldMapper, FieldDef, ExtractedRow, MappingResult } from "./field-mapper";

interface UploadProps {
    onCancel: () => void;
}

// Sample admin import schema
const ADMIN_FIELDS: FieldDef[] = [
    {
        key: "full_name",
        label: "Full name",
        aliases: ["name", "first name", "first_name", "full name"],
        required: true,
    },
    {
        key: "email",
        label: "Email",
        aliases: ["e-mail", "mail", "email address"],
        required: true,
        validator: (value) => {
            if (typeof value !== "string" || !value.includes("@")) return "Invalid email";
            return null;
        },
    },
    {
        key: "role",
        label: "Role",
        aliases: ["user role", "permission", "access"],
        required: true,
    },
    {
        key: "school_slug",
        label: "School slug",
        aliases: ["school", "academy", "institution"],
        required: false,
    },
];

export function Upload({ onCancel }: UploadProps) {
    const [extractedRows, setExtractedRows] = React.useState<ExtractedRow[]>([]);
    const [mappingResult, setMappingResult] = React.useState<MappingResult | null>(null);

    const handleUploaded = (raw: Record<string, object>[]) => {
        const withIds: ExtractedRow[] = raw.map((r, i) => ({
            __id:
                typeof crypto !== "undefined" && crypto.randomUUID
                    ? crypto.randomUUID()
                    : `row-${i}`,
            ...r,
        })) as ExtractedRow[];
        setExtractedRows(withIds);
        setMappingResult(null);
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
                        desiredFields={ADMIN_FIELDS}
                        onMappingChange={setMappingResult}
                    />
                )}
            </div>
        </div>
    );
}
