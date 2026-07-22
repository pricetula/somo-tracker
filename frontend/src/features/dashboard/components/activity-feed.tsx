/**
 * ActivityFeed — a chronological list of recent actions in the school.
 *
 * Shows recent import jobs (student imports, staff invitations) as
 * a lightweight activity feed. Hidden when there's no activity.
 */

"use client";

import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";
import { format } from "date-fns";
import { useDashboardRecentActivity } from "../hooks/use-dashboard-summary";
import { UploadIcon, MailIcon } from "lucide-react";

function ActivityIcon({ type }: { type: string }) {
    if (type === "staff_invite") {
        return <MailIcon className="size-4 shrink-0" />;
    }
    return <UploadIcon className="size-4 shrink-0" />;
}

export function ActivityFeed() {
    const { data: items, isLoading } = useDashboardRecentActivity();

    if (isLoading) {
        return (
            <section>
                <h2 className="mb-3 text-lg font-medium">Recent Activity</h2>
                <div className="space-y-3">
                    <Skeleton className="h-14 w-full" />
                    <Skeleton className="h-14 w-full" />
                    <Skeleton className="h-14 w-full" />
                </div>
            </section>
        );
    }

    if (!items || items.length === 0) {
        return null;
    }

    return (
        <section>
            <h2 className="mb-3 text-lg font-medium">Recent Activity</h2>
            <div className="space-y-2">
                {items.map((item) => (
                    <Link
                        key={item.id}
                        href={item.href ?? "#"}
                        className="hover:bg-muted/50 flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
                    >
                        <ActivityIcon type={item.type} />
                        <div className="flex min-w-0 flex-1 items-center justify-between gap-4">
                            <div className="min-w-0">
                                <p className="text-sm font-medium">{item.label}</p>
                                <p className="text-muted-foreground text-sm">{item.description}</p>
                            </div>
                            <time
                                className="text-muted-foreground shrink-0 text-xs tabular-nums"
                                dateTime={item.timestamp}
                            >
                                {format(new Date(item.timestamp), "MMM d, HH:mm")}
                            </time>
                        </div>
                    </Link>
                ))}
            </div>
        </section>
    );
}
