/**
 * MemberEditDialog — edit a member's name (admins, nurses, finance).
 *
 * Reusable across admins, nurses, and finance listing pages.
 * Uses getMember / updateMember from the shared members API.
 */

"use client";

import { useState, useEffect } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
    DialogClose,
} from "@/components/ui/dialog";
import { getMember, updateMember } from "@/lib/api/members";
import { getErrorMessage } from "@/lib/errors";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

interface MemberEditDialogProps {
    userId: string | null;
    open: boolean;
    onOpenChange: (open: boolean) => void;
    /** Query key to invalidate on successful save. */
    invalidationKey: string[];
}

export function MemberEditDialog({
    userId,
    open,
    onOpenChange,
    invalidationKey,
}: MemberEditDialogProps) {
    const queryClient = useQueryClient();

    const {
        data: member,
        isLoading,
        isError,
        error,
    } = useQuery({
        queryKey: ["member", userId],
        queryFn: () => getMember(userId!),
        enabled: !!userId && open,
    });

    const updateMutation = useMutation({
        mutationFn: (payload: { full_name: string }) => updateMember(userId!, payload),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: invalidationKey });
            toast.success("Member updated");
            onOpenChange(false);
        },
        onError: (err) => {
            toast.error(getErrorMessage(err));
        },
    });

    const [fullName, setFullName] = useState("");

    // Reset form when member data loads — syncs external API data into local form state.
    // This is a deliberate initialization effect, not cascading state updates.
    useEffect(() => {
        // eslint-disable-next-line react-hooks/set-state-in-effect
        if (member) setFullName(member.full_name ?? "");
    }, [member]);

    function handleSave() {
        if (!userId || !fullName.trim()) return;
        updateMutation.mutate({ full_name: fullName.trim() });
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Edit Member</DialogTitle>
                </DialogHeader>

                {isLoading ? (
                    <div className="space-y-3 py-4">
                        <Skeleton className="h-5 w-32" />
                        <Skeleton className="h-9 w-full" />
                        <Skeleton className="h-5 w-24" />
                        <Skeleton className="h-9 w-48" />
                    </div>
                ) : isError ? (
                    <Alert variant="destructive">
                        <AlertDescription>{getErrorMessage(error)}</AlertDescription>
                    </Alert>
                ) : member ? (
                    <div className="space-y-4 py-2">
                        <div className="space-y-1.5">
                            <Label>Email</Label>
                            <p className="text-muted-foreground">{member.email}</p>
                        </div>
                        <div className="space-y-1.5">
                            <Label>Full Name</Label>
                            <Input
                                value={fullName}
                                onChange={(e) => setFullName(e.target.value)}
                                placeholder="Full name"
                            />
                        </div>
                        {updateMutation.error && (
                            <p className="text-destructive">
                                {getErrorMessage(updateMutation.error)}
                            </p>
                        )}
                    </div>
                ) : (
                    <p className="text-muted-foreground py-4">Member not found.</p>
                )}

                <DialogFooter>
                    <DialogClose asChild>
                        <Button variant="outline">Cancel</Button>
                    </DialogClose>
                    <Button
                        onClick={handleSave}
                        disabled={updateMutation.isPending || !member || !fullName.trim()}
                    >
                        {updateMutation.isPending ? "Saving…" : "Save"}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
