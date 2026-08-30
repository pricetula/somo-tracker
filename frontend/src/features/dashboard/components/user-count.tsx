"use client";

import React from "react";
import Link from "next/link";
import { numberCompactor } from "@/lib/number-compactor";
import { useMemberCounts } from "@/features/dashboard/hooks/use-member-counts";

export function UserCount() {
    const { data: counts, isLoading, isError, error } = useMemberCounts();

    const memberCountList = React.useMemo(
        () => [
            {
                count: counts?.admins ?? 0,
                label: `Admin${counts?.admins === 1 ? "" : "s"}`,
                url: "/admins",
            },
            {
                count: counts?.students ?? 0,
                label: `Student${counts?.students === 1 ? "" : "s"}`,
                url: "/students",
            },
            // {
            //     count: counts?.nurses ?? 0,
            //     label: `Nurse${counts?.nurses === 1 ? "" : "s"}`,
            //  url: "/admins"
            // },
            {
                count: counts?.teachers ?? 0,
                label: `Teacher${counts?.teachers === 1 ? "" : "s"}`,
                url: "/teachers",
            },
            // {
            //     count: counts?.parents ?? 0,
            //     label: `Parent${counts?.parents === 1 ? "" : "s"}`,
            //     url: "/parents"
            // },
            // {
            //     count: counts?.finance ?? 0,
            //     label: "Finance",
            //     url: "/finance"
            // },
        ],
        [counts]
    );

    if (isLoading) {
        return <section>Loading...</section>;
    }

    if (isError) {
        return <section>Error: {error?.message}</section>;
    }

    return (
        <section className="flex items-center gap-4">
            {memberCountList.map((m) => (
                <Link
                    href={m.url}
                    className="flex flex-col items-center no-underline!"
                    key={m.label}
                >
                    <span className="text-2xl">{numberCompactor(m.count)}</span>
                    <span className="text-muted-foreground">{m.label}</span>
                </Link>
            ))}
        </section>
    );
}
