"use client";

import React, { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";
import { getErrorMessage } from "@/lib/errors";

// ─── Types ────────────────────────────────────────────────────────────────

interface ErrorBoundaryProps {
    children: ReactNode;
    /** Optional fallback UI. If not provided, a default graceful UI is shown. */
    fallback?: ReactNode;
    /** Optional callback for reporting errors (e.g. Sentry). */
    onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

// ─── Component ────────────────────────────────────────────────────────────

/**
 * A React ErrorBoundary that wraps major routes and features.
 *
 * Distinguishes between:
 * - ApiError (operational, e.g. 404): shows a graceful message using
 *   error.message from the backend.
 * - Unexpected errors (programming bugs): reports to the error tracker
 *   (via onError prop) and shows a generic "Something went wrong" message.
 *
 * Do not rely solely on the global error.tsx — every major route must have
 * its own ErrorBoundary.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
        // Report to error tracker via the optional callback
        this.props.onError?.(error, errorInfo);

        // For unexpected (non-ApiError) errors, also log to console
        if (!(error instanceof ApiError)) {
            console.error("[ErrorBoundary] unexpected error:", {
                name: error.name,
                message: error.message,
                componentStack: errorInfo.componentStack,
            });
        }
    }

    render(): ReactNode {
        if (this.state.hasError && this.state.error) {
            // If a custom fallback is provided, render it
            if (this.props.fallback) {
                return this.props.fallback;
            }

            const error = this.state.error;
            const isApiError = error instanceof ApiError;

            // Operational error (ApiError): show the backend's message
            if (isApiError) {
                return (
                    <div className="flex flex-col items-center justify-center gap-4 p-8 text-center">
                        <h2 className="text-2xl font-semibold tracking-tight">
                            {getErrorMessage(error)}
                        </h2>
                        <p className="text-muted-foreground">
                            The server returned an error. Please try again.
                        </p>
                        <Button
                            onClick={() => this.setState({ hasError: false, error: null })}
                            variant="default"
                        >
                            Try again
                        </Button>
                    </div>
                );
            }

            // Programming error (unexpected): show generic message, no internal details
            return (
                <div className="flex flex-col items-center justify-center gap-4 p-8 text-center">
                    <h2 className="text-2xl font-semibold tracking-tight">Something went wrong</h2>
                    <p className="text-muted-foreground">
                        An unexpected error occurred. Our team has been notified.
                    </p>
                    <Button
                        onClick={() => this.setState({ hasError: false, error: null })}
                        variant="default"
                    >
                        Try again
                    </Button>
                </div>
            );
        }

        return this.props.children;
    }
}
