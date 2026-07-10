/**
 * Intercepted route for parent bulk import via the modal slot.
 *
 * Renders a dialog overlay. The bulk import form will live here once
 * implemented — following the students import pattern (ImportJob system).
 */

"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Upload } from "lucide-react";

export default function ParentsImportModal() {
    const router = useRouter();

    return (
        <Dialog
            open
            onOpenChange={(open) => {
                if (!open) router.back();
            }}
        >
            <DialogContent>
                <DialogHeader>
                    <DialogTitle>Bulk Import Parents</DialogTitle>
                    <DialogDescription>
                        Add multiple parents at once. Bulk import will be available soon.
                    </DialogDescription>
                </DialogHeader>
                <div className="text-muted-foreground flex flex-col items-center gap-4 py-12">
                    <div className="bg-muted/50 flex size-12 items-center justify-center rounded-full">
                        <Upload className="text-muted-foreground size-5" />
                    </div>
                    <div className="text-center">
                        <p className="font-medium">Coming soon</p>
                        <p className="text-muted-foreground mt-0.5 text-xs">
                            You&apos;ll be able to import parents via CSV upload or manual entry.
                        </p>
                    </div>
                    <Button variant="outline" size="sm" asChild>
                        <Link href="/parents/new" onClick={() => router.back()}>
                            Add a single parent instead
                        </Link>
                    </Button>
                </div>
            </DialogContent>
        </Dialog>
    );
}
