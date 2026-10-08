"use client";

// import Link from "next/link";
import { useMemo } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { useUserCounts } from "../hooks/use-user-counts";
import type { UserCounts } from "../types/user-counts";

export function UserCountsPanel() {
    const { data, isLoading, isError } = useUserCounts();

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

    return <section className="space-y-6">{JSON.stringify(rows)}</section>;
}
