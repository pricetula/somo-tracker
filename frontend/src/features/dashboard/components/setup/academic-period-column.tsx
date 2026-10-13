"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";

type Props = {
    exists: boolean;
    yearName?: string;
    termName?: string;
    status?: string;
};

export function AcademicPeriodColumn({ exists, yearName, termName, status }: Props) {
    if (!exists) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Academic Period</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                    <p className="text-muted-foreground text-sm">
                        Define the operational timeline before building structures.
                    </p>
                    <Link href="/settings/academic-period/add" className={buttonVariants()}>
                        Define Academic Year & Term
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Academic Period</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
                <Badge variant="secondary">{yearName}</Badge>
                <Badge variant="outline">{termName}</Badge>
                {status && <p className="text-muted-foreground text-xs">Status: {status}</p>}
            </CardContent>
        </Card>
    );
}
