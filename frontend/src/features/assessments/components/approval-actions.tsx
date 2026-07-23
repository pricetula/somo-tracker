/**
 * ApprovalActions — Admin approve/reject workflow controls for a session.
 *
 * Shows Approve and Reject buttons when the session is PENDING_APPROVAL.
 * On reject, requires a comment explaining why.
 */

"use client";

import { useState } from "react";
import { CheckCircle, XCircle, Loader2 } from "lucide-react";

import { useApproveSession, useRejectSession } from "../hooks/use-assessments";
import { getErrorMessage } from "@/lib/errors";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
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
import { toast } from "sonner";

interface Props {
    sessionId: string;
    status: string;
}

export function ApprovalActions({ sessionId, status }: Props) {
    const approveMutation = useApproveSession();
    const rejectMutation = useRejectSession();
    const [rejectionComment, setRejectionComment] = useState("");

    if (status !== "PENDING_APPROVAL") return null;

    const handleApprove = () => {
        approveMutation.mutate(sessionId, {
            onError: (err) => toast.error(getErrorMessage(err)),
        });
    };

    const handleReject = () => {
        if (!rejectionComment.trim()) {
            toast.error("Please provide a reason for rejection.");
            return;
        }
        rejectMutation.mutate(
            { id: sessionId, rejection_comment: rejectionComment.trim() },
            {
                onSuccess: () => setRejectionComment(""),
                onError: (err) => toast.error(getErrorMessage(err)),
            }
        );
    };

    return (
        <div className="flex items-center gap-2">
            {/* Approve button */}
            <Button
                size="sm"
                className="bg-emerald-600 text-white hover:bg-emerald-700"
                onClick={handleApprove}
                disabled={approveMutation.isPending}
            >
                {approveMutation.isPending ? (
                    <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                ) : (
                    <CheckCircle className="mr-1.5 h-4 w-4" />
                )}
                Approve & Publish
            </Button>

            {/* Reject dialog */}
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button
                        variant="outline"
                        size="sm"
                        className="text-destructive border-destructive/30 hover:bg-destructive/10"
                        disabled={rejectMutation.isPending}
                    >
                        <XCircle className="mr-1.5 h-4 w-4" />
                        Reject
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Reject Assessment</AlertDialogTitle>
                        <AlertDialogDescription>
                            Provide feedback so the teacher can correct the issues and re-submit.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <div className="space-y-2">
                        <Label htmlFor="rejection-comment">Rejection Reason</Label>
                        <Textarea
                            id="rejection-comment"
                            value={rejectionComment}
                            onChange={(e) => setRejectionComment(e.target.value)}
                            placeholder="Please review Kamau's score — it looks like a typo."
                            rows={3}
                        />
                    </div>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={(e) => {
                                e.preventDefault();
                                handleReject();
                            }}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            {rejectMutation.isPending ? "Rejecting..." : "Reject & Return to Draft"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
