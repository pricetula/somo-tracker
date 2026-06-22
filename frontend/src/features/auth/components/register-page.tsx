/**
 * Register Page — page-level wrapper that provides Suspense for useSearchParams.
 */

"use client";

import { Suspense } from "react";
import { Loader2 } from "lucide-react";

import { RegisterForm } from "./register-form";

// ─── Types ─────────────────────────────────────────────────────────────────

export interface RegisterPageProps {
    tooltipSummary?: string;
}

// ─── Component ─────────────────────────────────────────────────────────────

export function RegisterPage({ tooltipSummary }: RegisterPageProps) {
    return (
        <Suspense
            fallback={
                <div className="flex min-h-screen items-center justify-center">
                    <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
                </div>
            }
        >
            <RegisterForm tooltipSummary={tooltipSummary} />
        </Suspense>
    );
}
