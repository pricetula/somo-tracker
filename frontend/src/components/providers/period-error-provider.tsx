"use client";

import React, { createContext, useCallback, useContext, useState } from "react";
import { useRouter } from "next/navigation";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog";

interface PeriodErrorContextValue {
    showPeriodError: (
        retry?: () => Promise<void>,
        options?: { showUseUpcoming?: boolean; redirectTo?: string }
    ) => void;
}

const PeriodErrorContext = createContext<PeriodErrorContextValue | undefined>(undefined);

export function PeriodErrorProvider({ children }: { children: React.ReactNode }) {
    const [open, setOpen] = useState(false);
    const [retryFn, setRetryFn] = useState<(() => Promise<void>) | null>(null);
    const [showUseUpcoming, setShowUseUpcoming] = useState(true);
    const [redirectTo, setRedirectTo] = useState<string | null>(null);
    const router = useRouter();

    const showPeriodError = useCallback(
        (
            retry?: () => Promise<void>,
            options?: { showUseUpcoming?: boolean; redirectTo?: string }
        ) => {
            if (retry) setRetryFn(() => retry);
            setShowUseUpcoming(options?.showUseUpcoming ?? true);
            setRedirectTo(options?.redirectTo ?? null);
            setOpen(true);
        },
        []
    );

    const handleCreatePeriod = () => {
        setOpen(false);
        router.push("/settings/academic-period/add");
    };

    const handleRetry = async () => {
        setOpen(false);
        if (retryFn) {
            try {
                await retryFn();
                if (redirectTo) {
                    router.push(redirectTo);
                }
            } catch (e) {
                console.error(e);
            }
        }
    };

    return (
        <PeriodErrorContext.Provider value={{ showPeriodError }}>
            {children}
            <AlertDialog open={open} onOpenChange={setOpen}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>No active academic term</AlertDialogTitle>
                        <AlertDialogDescription>
                            There is no active academic term to complete this action in. Create a
                            new academic period or use an upcoming period if available.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel onClick={() => setOpen(false)}>Cancel</AlertDialogCancel>
                        <AlertDialogAction onClick={handleCreatePeriod}>
                            Create new period
                        </AlertDialogAction>
                        {showUseUpcoming && (
                            <AlertDialogAction onClick={handleRetry}>
                                Use upcoming period
                            </AlertDialogAction>
                        )}
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </PeriodErrorContext.Provider>
    );
}

export function usePeriodError() {
    const ctx = useContext(PeriodErrorContext);
    if (!ctx) throw new Error("usePeriodError must be used within PeriodErrorProvider");
    return ctx;
}
