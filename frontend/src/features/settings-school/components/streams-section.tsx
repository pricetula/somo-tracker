"use client";

import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useStreamList } from "@/features/streams";
import Link from "next/link";

import { StreamRow } from "./stream-row";

export function StreamsSection() {
    const { data, isLoading, isError, error } = useStreamList();

    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <div>
                    <h2 className="text-foreground text-lg font-semibold">Streams</h2>
                    <p className="text-muted-foreground">
                        Manage the streams (sections) available in your school.
                    </p>
                </div>
                <Button variant="outline" asChild>
                    <Link href="/streams/add">
                        <Plus className="mr-1 h-4 w-4" />
                        Add Stream
                    </Link>
                </Button>
            </div>

            {isLoading ? (
                <div className="space-y-2">
                    <Skeleton className="h-10 w-full" />
                    <Skeleton className="h-10 w-full" />
                    <Skeleton className="h-10 w-full" />
                </div>
            ) : isError ? (
                <p className="text-destructive">{error?.message ?? "Failed to load streams."}</p>
            ) : data && data.items.length === 0 ? (
                <p className="text-muted-foreground">No streams yet. Add one to get started.</p>
            ) : (
                <div className="divide-border/50 border-border/50 divide-y rounded-md border">
                    {data?.items.map((stream) => (
                        <StreamRow key={stream.id} stream={stream} />
                    ))}
                </div>
            )}
        </div>
    );
}
