/**
 * CreateScaleProfileForm — Creates a new grading scale profile.
 *
 * After creation, the admin is navigated to the profile detail page to
 * set up percentage ranges.
 */

"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";

import { createScaleProfile } from "@/lib/api/assessments";
import type { ScaleProfile } from "@/lib/api/assessments";
import { getErrorMessage, isApiError } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface Props {
    onSuccess?: (profile: ScaleProfile) => void;
}

export function CreateScaleProfileForm({ onSuccess }: Props) {
    const router = useRouter();
    const [name, setName] = useState("");
    const [error, setError] = useState<string | null>(null);

    const createMutation = useMutation({
        mutationFn: () => {
            if (!name.trim()) throw new Error("Profile name is required.");
            return createScaleProfile({ name: name.trim() });
        },
        onSuccess: (result) => {
            // Navigate to profile detail to set up ranges
            router.push(`/grading/${result.id}`);
            onSuccess?.(result as unknown as ScaleProfile);
        },
        onError: (err) => {
            if (isApiError(err) && err.status === 400 && err.errors) {
                setError(err.errors.name?.[0] ?? err.message);
            } else {
                setError(getErrorMessage(err));
            }
        },
    });

    const handleSubmit = useCallback(
        (e: React.FormEvent) => {
            e.preventDefault();
            setError(null);
            createMutation.mutate();
        },
        [createMutation]
    );

    return (
        <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
                <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                </Alert>
            )}

            <div className="space-y-1.5">
                <Label htmlFor="name">Profile Name</Label>
                <Input
                    id="name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder='e.g. "Grade 4 Standard Conversion"'
                    disabled={createMutation.isPending}
                />
                <p className="text-muted-foreground text-xs">
                    Give this scale a descriptive name. Once created, the name cannot be changed.
                </p>
            </div>

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
                <Button type="submit" size="sm" disabled={createMutation.isPending || !name.trim()}>
                    {createMutation.isPending ? (
                        <>
                            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                            Creating...
                        </>
                    ) : (
                        "Create Profile"
                    )}
                </Button>
            </div>
        </form>
    );
}
