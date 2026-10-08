"use client";

// import Link from "next/link";
import { useMemo } from "react";
import { CircleQuestionMark, Plus } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { numberCompactor } from "@/lib/number-compactor";
import { useUserCounts } from "../hooks/use-user-counts";
import type { UserCounts } from "../types/user-counts";
import Link from "next/link";

export function UserCountsPanel() {
    const { data, isLoading, isError } = useUserCounts();

    const counts = data as UserCounts | undefined;

    const rows = useMemo(() => {
        const students = counts?.students || 0;
        const smale = counts?.students_male || 0;
        const sfemale = counts?.students_female || 0;
        const smalePercentage = students > 0 ? ((smale / students) * 100).toFixed(2) : 0;
        const sfemalePercentage = students > 0 ? ((sfemale / students) * 100).toFixed(2) : 0;
        const guardians = counts?.guardians || 0;
        const teachers = counts?.teachers || 0;
        const finance = counts?.finance || 0;
        const admins = counts?.admins || 0;
        return [
            {
                label: `Student${students !== 1 ? "s" : ""}`,
                count: students,
                tip: (
                    <>
                        <span>
                            This shows the total number of students at the school currently{" "}
                            {students}
                        </span>
                        <br />
                        <span className="flex items-center gap-4">
                            <span>
                                Male <b>{smalePercentage}%</b>
                            </span>

                            <span>
                                Female <b>{sfemalePercentage}%</b>
                            </span>
                        </span>
                    </>
                ),
                addHref: "/students/add",
                viewHref: "/students",
                addLabel: "Add Students",
            },
            {
                label: `Teacher${teachers !== 1 ? "s" : ""}`,
                count: teachers,
                tip: `This shows the total number of teachers at the school currently ${teachers}`,
                addHref: "/teachers/invite",
                viewHref: "/teachers",
                addLabel: "Invite Teachers",
            },
            {
                label: `Guardian${guardians !== 1 ? "s" : ""}`,
                count: guardians,
                tip: `This shows the total number of guardians at the school currently ${guardians}`,
                addHref: "/guardians/invite",
                viewHref: "/guardians",
                addLabel: "Invite Guardians",
            },
            {
                label: "Finance",
                count: finance,
                tip: `This shows the total number of finance at the school currently ${finance}`,
                addHref: "/finance/invite",
                viewHref: "/finance",
                addLabel: "Invite Finance",
            },
            {
                label: `Admin${admins !== 1 ? "s" : ""}`,
                count: admins,
                tip: `This shows the total number of admins at the school currently ${admins}`,
                addHref: "/admins/invite",
                viewHref: "/admins",
                addLabel: "Invite Admins",
            },
        ];
    }, [counts]);

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
        <section className="mb-4 grid grid-cols-2 gap-4 border-b border-dashed pb-4 md:grid-cols-3 lg:grid-cols-5">
            {rows.map((row) => (
                <div key={row.label} className="flex flex-col gap-2">
                    <div className="flex items-center gap-1">
                        <Link href={row.viewHref} className="space-x-1 text-lg">
                            <span>{numberCompactor(row.count)}</span>
                            <span className="text-muted-foreground">{row.label}</span>
                        </Link>
                        <Tooltip>
                            <TooltipTrigger
                                render={
                                    <CircleQuestionMark
                                        size={14}
                                        className="text-muted-foreground"
                                    />
                                }
                            />
                            <TooltipContent>
                                <p>{row.tip}</p>
                            </TooltipContent>
                        </Tooltip>
                    </div>
                    <Link href={row.addHref} className="text-muted-foreground flex gap-1">
                        <Plus size={14} />
                        <span>{row.addLabel}</span>
                    </Link>
                </div>
            ))}
        </section>
    );
}
