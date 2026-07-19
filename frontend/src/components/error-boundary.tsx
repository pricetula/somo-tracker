/**
 * Error boundary component.
 *
 * Wraps major routes/features to isolate crashes:
 * - ApiError (operational) → show error.message gracefully
 * - Other errors (programming bugs) → report to error tracker, show generic message
 *
 * Usage:
 *   <ErrorBoundary>
 *     <MyFeature />
 *   </ErrorBoundary>
 *
 * See frontend/AGENTS.md — Error Handling section.
 */

"use client";

import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertCircle, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api/client";

interface ErrorBoundaryProps {
    children: ReactNode;
    /** Optional custom fallback. Receives the error and a reset function. */
    fallback?: (props: { error: Error; reset: () => void }) => ReactNode;
}

interface ErrorBoundaryState {
    error: Error | null;
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { error: null };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { error };
    }

    componentDidCatch(error: Error, info: ErrorInfo) {
        // Report programming bugs to the error tracker
        if (!(error instanceof ApiError)) {
            console.error("[ErrorBoundary] unexpected error:", {
                name: error.name,
                message: error.message,
                stack: error.stack,
                componentStack: info.componentStack,
            });
        }
    }

    private handleReset = () => {
        this.setState({ error: null });
    };

    render() {
        if (this.state.error) {
            if (this.props.fallback) {
                return this.props.fallback({
                    error: this.state.error,
                    reset: this.handleReset,
                });
            }

            const isApiError = this.state.error instanceof ApiError;

            return (
                <div className="flex flex-col items-center justify-center gap-3 rounded-lg border p-8">
                    <AlertCircle className="text-destructive size-8" />
                    <div className="text-center">
                        <p className="font-semibold">
                            {isApiError ? this.state.error.message : "Something went wrong"}
                        </p>
                        <p className="text-muted-foreground mt-1 text-sm">
                            {isApiError
                                ? "The server returned an error. Please try again."
                                : "An unexpected error occurred in this section."}
                        </p>
                    </div>
                    <Button variant="outline" size="sm" onClick={this.handleReset}>
                        <RefreshCw className="mr-1.5 size-3.5" />
                        Retry
                    </Button>
                </div>
            );
        }

        return this.props.children;
    }
}
