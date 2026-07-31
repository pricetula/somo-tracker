"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { getActiveImportJob } from "@/lib/api/imports";
import Link from "next/link";

export function ActiveJobBanner() {
    const { data: activeData } = useQuery({
        queryKey: ["import-jobs", "active"],
        queryFn: () => getActiveImportJob(),
        staleTime: 15 * 1000,
        refetchInterval: (query) => {
            const d = query.state.data;
            if (!d?.active || !d.job) return false;
            const terminalStatuses = ["completed", "completed_with_errors", "failed", "cancelled"];
            if (terminalStatuses.includes(d.job.status)) return false;
            return 5_000;
        },
    });

    const activeJob = activeData?.active ? activeData.job : null;
    if (!activeJob) return null;

    return (
        <div className="bg-muted/30 mb-4 rounded-md px-4 py-3">
            <div className="flex items-center justify-between">
                <div className="space-y-1">
                    <p className="text-foreground font-medium">Active Import Job</p>
                    <p className="text-muted-foreground text-xs">
                        {activeJob.job_type.replace(/_/g, " ")} &mdash;{" "}
                        {activeJob.success_count + activeJob.failed_count} of{" "}
                        {activeJob.total_records} processed
                    </p>
                </div>
                <Button variant="outline" size="sm" asChild>
                    <Link href={`/imports/${activeJob.id}`}>View Details</Link>
                </Button>
            </div>
        </div>
    );
}
