/**
 * SeedDefaultButton — triggers seeding of the default CBC curriculum.
 */

"use client";

import * as React from "react";
import { Database } from "lucide-react";

import { Button } from "@/components/ui/button";
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

import { useSeedDefaultCBC } from "../hooks/use-seed-default-cbc";

export function SeedDefaultButton() {
    const [open, setOpen] = React.useState(false);
    const { mutate, isPending } = useSeedDefaultCBC();

    const handleConfirm = React.useCallback(() => {
        setOpen(false);
        mutate();
    }, [mutate]);

    return (
        <AlertDialog open={open} onOpenChange={setOpen}>
            <AlertDialogTrigger
                render={
                    <Button variant="outline" size="sm">
                        <Database className="mr-1.5 size-3.5" />
                        Seed Default CBC Curriculum
                    </Button>
                }
            />
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>Seed Default CBC Curriculum?</AlertDialogTitle>
                    <AlertDialogDescription>
                        This will import the complete CBC curriculum for all grades (PP1–G12) into
                        the current school. Existing curriculum data for each grade will be
                        replaced. This action cannot be undone.
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel>Cancel</AlertDialogCancel>
                    <AlertDialogAction onClick={handleConfirm} disabled={isPending}>
                        {isPending ? "Seeding…" : "Seed Curriculum"}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
