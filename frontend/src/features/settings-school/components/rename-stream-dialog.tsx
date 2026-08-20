"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
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
import { useUpdateStream } from "@/features/streams";
import { type Stream } from "@/features/streams";

import { ColorPicker } from "./color-picker";

export function RenameStreamDialog({ stream }: { stream: Stream }) {
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
