"use client";

import Link from "next/link";
import { Briefcase, Plus, Shield, UserCheck, UserPlus, Users } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { numberCompactor } from "@/lib/number-compactor";
import { useUserCounts } from "../hooks/use-user-counts";
import type { UserCounts } from "../types/user-counts";

interface Role {
    key: keyof Pick<UserCounts, "teachers" | "guardians" | "finance" | "admins">;
    label: string;
    href: string;
    addHref: string;
    addLabel: string;
    icon: LucideIcon;
}

const ROLES: Role[] = [
    {
        key: "teachers",
        label: "Teachers",
        href: "/teachers",
        addHref: "/teachers/invite",
        addLabel: "Invite",
        icon: UserCheck,
    },
    {
        key: "guardians",
        label: "Guardians",
        href: "/guardians",
        addHref: "/guardians/invite",
        addLabel: "Invite",
        icon: UserPlus,
    },
    {
        key: "finance",
        label: "Finance",
        href: "/finance",
        addHref: "/finance/invite",
        addLabel: "Invite",
        icon: Briefcase,
    },
    {
        key: "admins",
        label: "Admins",
        href: "/admins",
        addHref: "/admins/invite",
        addLabel: "Invite",
        icon: Shield,
    },
];

const pct = (part: number, total: number) => (total > 0 ? (part / total) * 100 : 0);

export function UserCountsPanel() {
    const { data, isLoading, isError } = useUserCounts();
    const counts = data as UserCounts | undefined;

    if (isLoading) {
        return (
            <section
                aria-label="People at your school"
                className="grid gap-4 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1.9fr)]"
                aria-busy="true"
            >
                <Card className="h-64 animate-pulse" />
                <div className="grid grid-cols-2 gap-4">
                    {[...Array(4)].map((_, i) => (
                        <Card key={i} className="h-30 animate-pulse" />
                    ))}
                </div>
            </section>
        );
    }

    if (isError || !counts) {
        return (
            <Alert variant="destructive">
                <AlertDescription>
                    Couldn&apos;t load user counts. Refresh the page to try again.
                </AlertDescription>
            </Alert>
        );
    }

    const students = counts.students || 0;
    const male = counts.students_male || 0;
    const female = counts.students_female || 0;
    const malePct = pct(male, students);
    const femalePct = pct(female, students);

    return (
        <section
            aria-label="People at your school"
            className="grid gap-3 lg:grid-cols-[minmax(0,1.1fr)_minmax(0,1.9fr)]"
        >
            {/* Students: featured card with gender breakdown */}
            <Card className="justify-between gap-2 p-3">
                <CardHeader className="pb-1">
                    <CardTitle className="flex items-center gap-1.5 text-xs">
                        <span className="bg-primary/10 text-primary flex size-6 items-center justify-center rounded">
                            <Users className="size-3.5" aria-hidden="true" />
                        </span>
                        Students
                    </CardTitle>
                </CardHeader>
                <CardContent className="space-y-1.5 pt-0">
                    <Link
                        href="/students"
                        aria-label={`View all ${students} students`}
                        className="hover:text-primary text-3xl font-semibold tracking-tight tabular-nums transition-colors"
                    >
                        {numberCompactor(students)}
                    </Link>
                    <p className="text-muted-foreground text-[11px]">enrolled this term</p>

                    {/* Gender split */}
                    <div className="space-y-1">
                        <div
                            className="bg-muted flex h-1 overflow-hidden rounded-full"
                            role="img"
                            aria-label={`${male} male, ${female} female`}
                        >
                            <div className="bg-primary" style={{ width: `${malePct}%` }} />
                            <div className="bg-teal-500" style={{ width: `${femalePct}%` }} />
                        </div>
                        <dl className="text-muted-foreground flex justify-between text-[10px]">
                            <div className="flex items-center gap-1">
                                <span
                                    className="bg-primary size-1 rounded-full"
                                    aria-hidden="true"
                                />
                                <dt className="font-medium">Male</dt>
                                <dd className="text-foreground font-medium tabular-nums">
                                    {male} ({malePct.toFixed(1)}%)
                                </dd>
                            </div>
                            <div className="flex items-center gap-1">
                                <span
                                    className="size-1 rounded-full bg-teal-500"
                                    aria-hidden="true"
                                />
                                <dt className="font-medium">Female</dt>
                                <dd className="text-foreground font-medium tabular-nums">
                                    {female} ({femalePct.toFixed(1)}%)
                                </dd>
                            </div>
                        </dl>
                    </div>

                    <Link
                        href="/students/add"
                        className="text-muted-foreground hover:text-primary inline-flex items-center gap-1 text-[11px] transition-colors"
                    >
                        <Plus size={10} aria-hidden="true" />
                        Add students
                    </Link>
                </CardContent>
            </Card>

            {/* Other roles: compact cards */}
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                {ROLES.map(({ key, label, href, addHref, addLabel, icon: Icon }) => {
                    const value = counts[key] || 0;

                    return (
                        <Card key={key} className="justify-between gap-1.5 p-2.5">
                            <CardHeader className="pb-1">
                                <CardTitle className="flex items-center gap-1.5 text-[11px]">
                                    <span className="bg-primary/10 text-primary flex size-5 items-center justify-center rounded">
                                        <Icon className="size-3" aria-hidden="true" />
                                    </span>
                                    {label}
                                </CardTitle>
                            </CardHeader>
                            <CardContent className="flex items-start justify-between space-y-1 pt-0">
                                <Link
                                    href={href}
                                    aria-label={`View all ${value} ${label.toLowerCase()}`}
                                    className="hover:text-primary text-2xl font-semibold tracking-tight tabular-nums transition-colors"
                                >
                                    {numberCompactor(value)}
                                </Link>
                                <Link
                                    href={addHref}
                                    aria-label={`${addLabel} ${label.toLowerCase()}`}
                                    className="text-muted-foreground hover:text-primary inline-flex items-center gap-1 text-[11px] transition-colors"
                                >
                                    <Plus size={10} aria-hidden="true" />
                                    {addLabel}
                                </Link>
                            </CardContent>
                        </Card>
                    );
                })}
            </div>
        </section>
    );
}
