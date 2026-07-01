/**
 * Create Parent Form — creates a parent/guardian profile.
 *
 * Fields: email (required), full_name (required), phone_number (required).
 * On success navigates to the parent detail page.
 */

"use client";

import * as React from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";

import { useCreateParent } from "../hooks/use-parents";
import { getErrorMessage } from "@/lib/errors";

// ─── Props ─────────────────────────────────────────────────────────────────

interface CreateParentFormProps {
    onSuccess?: (id: string) => void;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function CreateParentForm({ onSuccess }: CreateParentFormProps) {
    const router = useRouter();
    const createParent = useCreateParent();

    const [email, setEmail] = React.useState("");
    const [fullName, setFullName] = React.useState("");
    const [phoneNumber, setPhoneNumber] = React.useState("");
    const [error, setError] = React.useState<string | null>(null);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setError(null);

        if (!email.trim()) {
            setError("Email is required");
            return;
        }
        if (!fullName.trim()) {
            setError("Full name is required");
            return;
        }
        if (!phoneNumber.trim()) {
            setError("Phone number is required");
            return;
        }

        try {
            const result = await createParent.mutateAsync({
                email: email.trim(),
                full_name: fullName.trim(),
                phone_number: phoneNumber.trim(),
            });

            if (onSuccess) {
                onSuccess(result.id);
            } else {
                router.push(`/parents/${result.id}`);
            }
        } catch (err) {
            setError(getErrorMessage(err));
        }
    };

    const isSubmitting = createParent.isPending;

    return (
        <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
                <div className="text-destructive bg-destructive/10 rounded-md px-3 py-2 text-sm">
                    {error}
                </div>
            )}

            {/* Email */}
            <div className="space-y-1.5">
                <Label htmlFor="email">
                    Email <span className="text-destructive">*</span>
                </Label>
                <Input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="parent@example.com"
                    disabled={isSubmitting}
                />
                <p className="text-muted-foreground text-xs">
                    The user must already exist in the system. Their account will be linked as a
                    parent.
                </p>
            </div>

            {/* Full Name */}
            <div className="space-y-1.5">
                <Label htmlFor="full_name">
                    Full Name <span className="text-destructive">*</span>
                </Label>
                <Input
                    id="full_name"
                    value={fullName}
                    onChange={(e) => setFullName(e.target.value)}
                    placeholder="e.g. Jane Doe"
                    disabled={isSubmitting}
                />
            </div>

            {/* Phone Number */}
            <div className="space-y-1.5">
                <Label htmlFor="phone_number">
                    Phone Number <span className="text-destructive">*</span>
                </Label>
                <Input
                    id="phone_number"
                    value={phoneNumber}
                    onChange={(e) => setPhoneNumber(e.target.value)}
                    placeholder="e.g. +254712345678"
                    disabled={isSubmitting}
                />
                <p className="text-muted-foreground text-xs">
                    Used for M-Pesa billing notifications and SMS alerts.
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
                        "Create Parent"
                    )}
                </Button>
                <Button
                    type="button"
                    variant="ghost"
                    onClick={() => router.back()}
                    disabled={isSubmitting}
                >
                    Cancel
                </Button>
            </div>
        </form>
    );
}
