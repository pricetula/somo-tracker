/**
 * Timetable Track List — /timetable
 * Shows all timetable tracks for the active school.
 */
"use client";

import React from "react";
import Link from "next/link";
import { PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTracks } from "@/features/timetable/hooks";

export default function TimetableListPage() {
    const { data, isLoading } = useTracks();

    if (isLoading) {
        return (
            <div className="space-y-4 p-6">
                <h1 className="text-2xl font-semibold">Timetables</h1>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {[1, 2].map((i) => (
                        <div key={i} className="bg-muted h-40 animate-pulse rounded-xl border" />
                    ))}
                </div>
            </div>
        );
    }

    const tracks = data?.items ?? [];

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-semibold tracking-tight">Timetables</h1>
                    <p className="text-muted-foreground mt-1">Track schedules for your school.</p>
                </div>
                <Link href="/timetable/new">
                    <Button size="sm">
                        <PlusIcon className="mr-1.5 size-3.5" />
                        New Timetable
                    </Button>
                </Link>
            </div>

            {tracks.length === 0 ? (
                <div className="rounded-xl border p-12 text-center">
                    <p className="text-muted-foreground">No timetables yet.</p>
                    <Link href="/timetable/new" className="mt-4 inline-block">
                        <Button variant="outline" size="sm">
                            Create first timetable
                        </Button>
                    </Link>
                </div>
            ) : (
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {tracks.map((track) => (
                        <Link
                            key={track.id}
                            href={`/timetable/${track.id}`}
                            className="group bg-card rounded-xl border p-6 shadow-sm transition-shadow hover:shadow-md"
                        >
                            <h3 className="group-hover:text-primary text-lg font-semibold transition-colors">
                                {track.name}
                            </h3>
                            <p className="text-muted-foreground mt-1 truncate text-sm">
                                {track.description || "No description"}
                            </p>
                            <div className="text-muted-foreground mt-4 flex items-center gap-4 text-xs">
                                <span>Year: 2026</span>
                                <span>•</span>
                                <span>Active</span>
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
}
