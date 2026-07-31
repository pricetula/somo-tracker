"use client";

import { CheckCircle2, XCircle, AlertTriangle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { type StagedStudentRecord } from "./types";

export function StatusBadge({ status }: { status: StagedStudentRecord["status"] }) {
    switch (status) {
        case "valid":
            return (
                <Badge variant="default" className="bg-emerald-500/10 text-[10px] text-emerald-600">
                    <CheckCircle2 className="mr-0.5 size-3" />
                    Valid
                </Badge>
            );
        case "error":
            return (
                <Badge
                    variant="outline"
                    className="text-destructive border-destructive/30 text-[10px]"
                >
                    <XCircle className="mr-0.5 size-3" />
                    Error
                </Badge>
            );
        case "duplicate":
            return (
                <Badge variant="outline" className="border-amber-200 text-[10px] text-amber-600">
                    <AlertTriangle className="mr-0.5 size-3" />
                    Duplicate
                </Badge>
            );
        case "submitted":
            return (
                <Badge variant="outline" className="border-blue-200 text-[10px] text-blue-600">
                    Submitted
                </Badge>
            );
    }
}
