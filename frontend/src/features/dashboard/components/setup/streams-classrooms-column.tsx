"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import Link from "next/link";

type ClassroomItem = { id: string; name: string; grade: string; stream: string };

type Props = {
    locked: boolean;
    items?: ClassroomItem[];
    count?: number;
    streamsCount?: number;
};

export function StreamsClassroomsColumn({ locked, items, count = 0, streamsCount = 0 }: Props) {
    if (locked) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Streams & Classrooms</CardTitle>
                </CardHeader>
                <CardContent>
                    <p className="text-muted-foreground text-sm">Define academic period first.</p>
                </CardContent>
            </Card>
        );
    }

    if (streamsCount === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Streams & Classrooms</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                    <p className="text-muted-foreground text-sm">Define school streams first.</p>
                    <Link href="/settings/streams/add" className={buttonVariants()}>
                        Create Streams
                    </Link>
                </CardContent>
            </Card>
        );
    }

    if (!items || count === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle>Streams & Classrooms</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                    <p className="text-muted-foreground text-sm">
                        Build classroom containers for the active year.
                    </p>
                    <Link href="/classes/create-using-streams" className={buttonVariants()}>
                        Auto populate Classrooms
                    </Link>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle>Streams & Classrooms</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
                <div className="grid grid-cols-2 gap-3 text-center">
                    <div className="bg-muted rounded-lg p-3">
                        <div className="text-2xl font-bold">{count}</div>
                        <div className="text-muted-foreground text-xs">Classrooms</div>
                    </div>
                    <div className="bg-muted rounded-lg p-3">
                        <div className="text-2xl font-bold">{streamsCount}</div>
                        <div className="text-muted-foreground text-xs">Streams</div>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
