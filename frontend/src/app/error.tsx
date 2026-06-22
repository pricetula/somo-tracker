"use client";

import { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";

/**
 * Global error.tsx for the Next.js App Router.
 *
 * Distinguishes between:
 * - ApiError (operational): shows the error message from the backend.
 * - Unexpected errors (programming): shows a generic message and reports
 *   to the error tracker (console.error as a fallback).
 */
export default function GlobalError({
    error,
    reset,
}: {
    error: Error & { digest?: string };
    reset: () => void;
}) {
    useEffect(() => {
        // Report unexpected errors (programming bugs) to the error tracker
        if (!(error instanceof ApiError)) {
            console.error("[GlobalError] unexpected error:", {
                name: error.name,
                message: error.message,
                digest: error.digest,
                stack: error.stack,
            });
        }
    }, [error]);

    const isApiError = error instanceof ApiError;
    const title = isApiError ? error.message : "Something went wrong";
    const description = isApiError
        ? "The server returned an error. Please try again."
        : "An unexpected error occurred. Our team has been notified.";

    return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-8">
            <div className="text-center">
                <h1 className="text-4xl font-bold tracking-tight">{title}</h1>
                <p className="text-muted-foreground mt-2">{description}</p>
                {error.digest && (
                    <p className="text-muted-foreground mt-1 text-xs">Error ID: {error.digest}</p>
                )}
            </div>
            <Button onClick={reset} variant="default">
                Try again
            </Button>
        </div>
    );
}
