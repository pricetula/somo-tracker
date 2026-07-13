/**
 * AddStreamForm — creates a new stream.
 *
 * Used by both the StreamsSection inline dialog and the /streams/add intercepted route.
 */

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { createStream } from "@/lib/api/streams";
import { getErrorMessage } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { streamKeys } from "../hooks/use-streams";

// ─── Color Swatch ─────────────────────────────────────────────────────────

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
                    <span className="text-lg leading-none">+</span>
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

// ─── Props ─────────────────────────────────────────────────────────────────

interface AddStreamFormProps {
    /** Called after successful creation. */
    onSuccess?: () => void;
    /** Pre-populate the name field (e.g. from a query param like ?value=abc). */
    defaultValue?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function AddStreamForm({ onSuccess, defaultValue }: AddStreamFormProps) {
    const router = useRouter();
    const queryClient = useQueryClient();
    const [name, setName] = useState(defaultValue ?? "");
    const [color, setColor] = useState(COLOR_OPTIONS[0]);
    const [error, setError] = useState<string | null>(null);

    const createMutation = useMutation({
        mutationFn: () => createStream({ name: name.trim(), color }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: streamKeys.list() });
            setName("");
            setColor(COLOR_OPTIONS[0]);
            onSuccess?.();
            router.back();
        },
        onError: (err) => {
            setError(getErrorMessage(err));
        },
    });

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (!name.trim()) return;
        setError(null);
        createMutation.mutate();
    };

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            <div className="space-y-2">
                <Label htmlFor="stream-name">Name</Label>
                <Input
                    id="stream-name"
                    placeholder="e.g. Blue, Red, Green"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    autoFocus
                />
            </div>

            <ColorPicker value={color} onChange={setColor} />

            <div className="flex items-center justify-end gap-2 pt-2">
                <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => router.back()}
                    disabled={createMutation.isPending}
                >
                    Cancel
                </Button>
                <Button type="submit" size="sm" disabled={!name.trim() || createMutation.isPending}>
                    {createMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Adding…
                        </>
                    ) : (
                        "Add"
                    )}
                </Button>
            </div>
        </form>
    );
}
