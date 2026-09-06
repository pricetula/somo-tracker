/**
 * Register Page — redirects to login/dashboard since backend handles user
 * provisioning via Stytch magic-link callback.
 */

"use client";

import { Suspense } from "react";
import { Loader2 } from "lucide-react";

import { RegisterForm } from "./register-form";

export function RegisterPage() {
    return (
        <Suspense
            fallback={
                <div className="flex min-h-screen items-center justify-center">
                    <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
                </div>
            }
        >
            <RegisterForm />
        </Suspense>
    );
}
