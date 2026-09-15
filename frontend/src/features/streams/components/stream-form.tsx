"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useStream, useUpdateStream, useCreateStreams } from "../hooks/use-streams";
import { getErrorMessage } from "@/lib/errors";
import { toast } from "sonner";

interface StreamFormProps {
    streamId?: string;
    onSuccess?: () => void;
}

export function StreamForm({ streamId, onSuccess }: StreamFormProps) {
    const router = useRouter();
    const isEdit = !!streamId;
    const { data: stream, isLoading } = useStream(streamId || "");
    const update = useUpdateStream();
    const create = useCreateStreams();
    const [name, setName] = useState(stream?.name ?? "");
    const [color, setColor] = useState(stream?.color ?? "");

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!name.trim()) {
            toast.error("Name is required");
            return;
        }
        try {
            if (isEdit) {
                await update.mutateAsync({
                    id: streamId!,
                    data: { name: name.trim(), color: color || null },
                });
            } else {
                await create.mutateAsync([name.trim()]);
            }
            if (onSuccess) {
                onSuccess();
            } else {
                router.back();
            }
        } catch (err) {
            toast.error(getErrorMessage(err));
        }
    };

    if (isEdit && isLoading && !stream) {
        return <div className="text-muted-foreground text-sm">Loading stream…</div>;
    }

    return (
        <form key={stream?.id ?? streamId ?? "new"} onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
                <Label htmlFor="stream-name">Stream name</Label>
                <Input
                    id="stream-name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., Arts"
                />
            </div>
            <div className="space-y-2">
                <Label htmlFor="stream-color">Color</Label>
                <div className="flex items-center gap-3">
                    <input
                        id="stream-color"
                        type="color"
                        value={color || "#3b82f6"}
                        onChange={(e) => setColor(e.target.value)}
                        className="h-9 w-9 cursor-pointer rounded border-0 p-0"
                    />
                    <Input
                        value={color}
                        onChange={(e) => setColor(e.target.value)}
                        placeholder="#3b82f6"
                        className="flex-1"
                    />
                </div>
            </div>
            <div className="flex items-center gap-2 pt-2">
                <Button type="submit" disabled={update.isPending || create.isPending}>
                    {isEdit ? "Save changes" : "Add stream"}
                </Button>
            </div>
        </form>
    );
}
