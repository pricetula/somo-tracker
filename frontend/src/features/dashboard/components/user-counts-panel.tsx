"use client";

import Link from "next/link";
import { useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useUserCounts } from "../hooks/use-user-counts";
import type { UserCounts } from "../types/user-counts";

interface Props {
    schoolId: string;
}

export function UserCountsPanel({ schoolId }: Props) {
    const { data, isLoading, isError } = useUserCounts(schoolId);

    const counts = data as UserCounts | undefined;

    const rows = useMemo(
        () =>
            counts
                ? [
                      {
                          label: "Students",
                          count: counts.students,
                          sub: `Male ${counts.students_male} / Female ${counts.students_female}`,
                          addHref: "/students/add",
                          viewHref: "/students",
                          addLabel: "Add Students",
                      },
                      {
                          label: "Teachers",
                          count: counts.teachers,
                          sub: "",
                          addHref: "/teachers/invite",
                          viewHref: "/teachers",
                          addLabel: "Invite Teachers",
                      },
                      {
                          label: "Guardians",
                          count: counts.guardians,
                          sub: "",
                          addHref: "/guardians/invite",
                          viewHref: "/guardians",
                          addLabel: "Invite Guardians",
                      },
                      {
                          label: "Finance",
                          count: counts.finance,
                          sub: "",
                          addHref: "/finance/invite",
                          viewHref: "/finance",
                          addLabel: "Invite Finance",
                      },
                      {
                          label: "Admins",
                          count: counts.admins,
                          sub: "",
                          addHref: "/admins/invite",
                          viewHref: "/admins",
                          addLabel: "Invite Admins",
                      },
                  ]
                : [],
        [counts]
    );

    if (isLoading) {
        return <div className="space-y-4">Loading user counts…</div>;
    }

    if (isError || !counts) {
        return (
            <Alert variant="destructive">
                <AlertDescription>Failed to load user counts.</AlertDescription>
            </Alert>
        );
    }

    return (
        <section className="space-y-6">
            <h2 className="text-xl font-semibold">User counts – active</h2>
            <div className="space-y-4">
                {rows.map((r) => (
                    <div key={r.label} className="flex items-center justify-between gap-4">
                        <div>
                            <div className="font-medium">
                                {r.label}{" "}
                                <span className="ml-2 tabular-nums">
                                    {r.count.toLocaleString()}
                                </span>
                            </div>
                            {r.sub && <div className="text-muted-foreground text-sm">{r.sub}</div>}
                        </div>
                        <div className="flex items-center gap-2">
                            <Button asChild size="sm" variant="secondary">
                                <Link href={r.viewHref}>View</Link>
                            </Button>
                            <Button asChild size="sm">
                                <Link href={r.addHref}>{r.addLabel}</Link>
                            </Button>
                        </div>
                    </div>
                ))}
                <div className="text-muted-foreground pt-2 text-sm">
                    Total active users: {counts.total_users.toLocaleString()}
                </div>
            </div>
        </section>
    );
}
