/**
 * StreamsSection — manage school streams (rename, delete).
 *
 * Add stream has moved to /streams/add (intercepted modal).
 * The listing, rename, and delete actions remain inline.
 */

"use client";

import { useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { useStreamList, useUpdateStream, useDeleteStream } from "@/features/streams";
import type { Stream } from "@/features/streams";
import Link from "next/link";

// ─── Color Swatch ─────────────────────────────────────────────────────────

function ColorDot({ color }: { color: string }) {
    if (!color) return null;
    return (
        <span
            className="inline-block h-3.5 w-3.5 rounded-full"
            style={{ backgroundColor: color }}
            aria-hidden="true"
        />
    );
}

const COLOR_OPTIONS = [
    "#ef4444",
    "#f97316",
    "#eab308",
    "#22c55e",
    "#14b8a6",
    "#3b82f6",
    "#8b5cf6",
    "#ec4899",
    "#78716c",
    "#06b6d4",
];

function ColorPicker({ value, onChange }: { value: string; onChange: (color: string) => void }) {
    return (
        <div className="space-y-2">
            <Label>Colour</Label>
            <div className="flex flex-wrap gap-2">
                {COLOR_OPTIONS.map((c) => (
                    <button
                        key={c}
                        type="button"
                        onClick={() => onChange(c)}
                        className={`h-7 w-7 rounded-full border-2 transition-all ${
                            value === c
                                ? "border-foreground scale-110"
                                : "border-transparent hover:scale-110"
                        }`}
                        style={{ backgroundColor: c }}
                        aria-label={`Select colour ${c}`}
                    />
                ))}
                {/* Custom color input */}
                <label
                    className="border-muted-foreground/50 text-muted-foreground hover:border-foreground/50 flex h-7 w-7 cursor-pointer items-center justify-center rounded-full border-2 border-dashed text-xs"
                    aria-label="Custom colour"
                >
                    <Plus className="h-3 w-3" />
                    <input
                        type="color"
                        value={value && !COLOR_OPTIONS.includes(value) ? value : "#000000"}
                        onChange={(e) => onChange(e.target.value)}
                        className="sr-only"
                    />
                </label>
            </div>
        </div>
    );
}

// ─── Rename Stream Dialog ─────────────────────────────────────────────────

function RenameStreamDialog({ stream }: { stream: Stream }) {
    const [open, setOpen] = useState(false);
    const [name, setName] = useState(stream.name);
    const [color, setColor] = useState(stream.color);
    const updateStream = useUpdateStream();

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!name.trim()) return;
        await updateStream.mutateAsync({
            id: stream.id,
            name: name.trim(),
            color,
        });
        setOpen(false);
    };

    const isUnchanged = name.trim() === stream.name && color === stream.color;

    return (
        <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
                <Button size="icon" variant="ghost" className="h-8 w-8">
                    <Pencil className="h-4 w-4" />
                    <span className="sr-only">Rename {stream.name}</span>
                </Button>
            </DialogTrigger>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Edit Stream</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit}>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="edit-stream-name">Name</Label>
                            <Input
                                id="edit-stream-name"
                                value={name}
                                onChange={(e) => setName(e.target.value)}
                                autoFocus
                            />
                        </div>
                        <ColorPicker value={color} onChange={setColor} />
                    </div>
                    <DialogFooter>
                        <DialogClose asChild>
                            <Button type="button" variant="ghost">
                                Cancel
                            </Button>
                        </DialogClose>
                        <Button
                            type="submit"
                            disabled={!name.trim() || isUnchanged || updateStream.isPending}
                        >
                            {updateStream.isPending ? "Saving…" : "Save"}
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}

// ─── Delete Stream Alert ──────────────────────────────────────────────────

function DeleteStreamAlert({ stream }: { stream: Stream }) {
    const deleteStream = useDeleteStream();

    return (
        <AlertDialog>
            <AlertDialogTrigger asChild>
                <Button
                    size="icon"
                    variant="ghost"
                    className="text-destructive hover:text-destructive h-8 w-8"
                >
                    <Trash2 className="h-4 w-4" />
                    <span className="sr-only">Delete {stream.name}</span>
                </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Delete Stream</AlertDialogTitle>
                    <AlertDialogDescription>
                        Are you sure you want to delete the stream <strong>{stream.name}</strong>?
                        This action cannot be undone.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel disabled={deleteStream.isPending}>Cancel</AlertDialogCancel>
                    <AlertDialogAction
                        onClick={() => deleteStream.mutate(stream.id)}
                        disabled={deleteStream.isPending}
                        className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    >
                        {deleteStream.isPending ? "Deleting…" : "Delete"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}

// ─── Stream Row ───────────────────────────────────────────────────────────

function StreamRow({ stream }: { stream: Stream }) {
    return (
        <div className="hover:bg-muted/50 flex items-center justify-between gap-4 rounded-md px-3 py-2">
            <div className="flex items-center gap-3">
                <ColorDot color={stream.color} />
                <span className="text-foreground font-medium">{stream.name}</span>
            </div>
            <div className="flex items-center gap-1">
                <RenameStreamDialog stream={stream} />
                <DeleteStreamAlert stream={stream} />
            </div>
        </div>
    );
}

// ─── Streams Section (exported) ───────────────────────────────────────────

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
