"use client";

import { Badge } from "@/components/ui/badge";

export function StatusBadge({ status }: { status: string }) {
    switch (status) {
        case "PENDING_REVIEW":
            return (
                <Badge variant="outline" className="text-amber-600">
                    Pending Review
                </Badge>
            );
        case "APPROVED":
            return (
                <Badge className="bg-green-100 text-green-700 hover:bg-green-100">Approved</Badge>
            );
        case "REJECTED":
            return <Badge variant="destructive">Rejected</Badge>;
        case "INCLUDED_IN_REPORT":
            return <Badge className="bg-sky-100 text-sky-700 hover:bg-sky-100">In Report</Badge>;
        default:
            return <Badge variant="outline">{status}</Badge>;
    }
}
