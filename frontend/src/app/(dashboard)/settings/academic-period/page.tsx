"use client";

import Link from "next/link";
import { buttonVariants } from "@/components/ui/button";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";

const mockTerms = [
    { id: "1", name: "Term 1", startDate: "2026-01-15", endDate: "2026-04-15", status: "upcoming" },
    { id: "2", name: "Term 2", startDate: "2026-05-01", endDate: "2026-08-15", status: "upcoming" },
    { id: "3", name: "Term 3", startDate: "2026-09-01", endDate: "2026-10-30", status: "upcoming" },
];

export default function AcademicPeriodPage() {
    const currentYear = 2026;

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold">Academic Year {currentYear}</h1>
                <Link
                    href="/settings/academic-period/add"
                    className={buttonVariants({ variant: "default" })}
                >
                    Add Period
                </Link>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Academic Terms</CardTitle>
                </CardHeader>
                <CardContent>
                    <Table>
                        <TableHeader>
                            <TableRow>
                                <TableHead>Name</TableHead>
                                <TableHead>Start Date</TableHead>
                                <TableHead>End Date</TableHead>
                                <TableHead>Status</TableHead>
                            </TableRow>
                        </TableHeader>
                        <TableBody>
                            {mockTerms.map((term) => (
                                <TableRow key={term.id}>
                                    <TableCell>{term.name}</TableCell>
                                    <TableCell>{term.startDate}</TableCell>
                                    <TableCell>{term.endDate}</TableCell>
                                    <TableCell>{term.status}</TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </CardContent>
            </Card>
        </div>
    );
}
