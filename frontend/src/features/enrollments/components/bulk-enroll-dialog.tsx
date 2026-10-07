"use client";

import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

export function BulkEnrollDialog({
    open,
    onClose,
    children,
}: {
    open: boolean;
    onClose: () => void;
    children?: React.ReactNode;
}) {
    return (
        <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
            <DialogContent className="sm:max-w-lg">
                <DialogHeader>
                    <DialogTitle>Bulk Enroll</DialogTitle>
                </DialogHeader>
                <div className="py-4">{children}</div>
                <DialogFooter>
                    <Button variant="outline" onClick={onClose}>
                        Close
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
