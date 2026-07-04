/**
 * Create School Form — creates a new school in the tenant.
 *
 * Fields: name (required).
 * On success, invalidates the schools list and me query so the new school
 * appears immediately in the switcher.
 */

"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";

import { useCreateSchool } from "../hooks/use-schools";
import { getErrorMessage, isApiError } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateSchoolFormProps {
    onSuccess?: (id: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateSchoolForm({ onSuccess }: CreateSchoolFormProps) {
    const createSchool = useCreateSchool();

    const [name, setName] = React.useState("");
    const [error, setError] = React.useState<string | null>(null);
    const [fieldErrors, setFieldErrors] = React.useState<Record<string, string> | null>(null);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);
        setFieldErrors(null);

        const trimmedName = name.trim();

        if (!trimmedName) {
            setFieldErrors({ name: "School name is required" });
            return;
        }

        if (trimmedName.length < 2) {
            setFieldErrors({ name: "School name must be at least 2 characters" });
            return;
        }

        try {
            const result = await createSchool.mutateAsync({ name: trimmedName });

            if (onSuccess) {
                onSuccess(result.id);
            }
        } catch (err) {
            if (isApiError(err) && err.status === 400 && err.errors) {
                // Map field-level validation errors
                const fieldMap: Record<string, string> = {};
                for (const [key, messages] of Object.entries(err.errors)) {
                    fieldMap[key] = messages[0];
                }
                setFieldErrors(fieldMap);
            } else {
                setError(getErrorMessage(err));
            }
        }
    };

    const isSubmitting = createSchool.isPending;

    return (
        <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* School Name */}
            <div className="space-y-1.5">
                <Label htmlFor="name">
                    School Name <span className="text-destructive">*</span>
                </Label>
                <Input
                    id="name"
                    value={name}
                    onChange={(e) => {
                        setName(e.target.value);
                        if (fieldErrors?.name) {
                            setFieldErrors((prev) => {
                                if (!prev) return null;
                                const next = { ...prev };
                                delete next.name;
                                return Object.keys(next).length ? next : null;
                            });
                        }
                    }}
                    placeholder="e.g. Moi Girls School"
                    disabled={isSubmitting}
                    aria-invalid={!!fieldErrors?.name}
                />
                {fieldErrors?.name && (
                    <p className="text-destructive text-xs">{fieldErrors.name}</p>
                )}
                <p className="text-muted-foreground text-xs">
                    A new school will be created within your tenant. CBC curriculum will be seeded
                    automatically.
                </p>
            </div>

            {/* Submit */}
            <div className="flex items-center gap-3 pt-2">
                <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting ? (
                        <>
                            <Loader2 className="mr-1.5 size-4 animate-spin" />
                            Creating…
                        </>
                    ) : (
                        "Create School"
                    )}
                </Button>
            </div>
        </form>
    );
}
