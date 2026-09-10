/**
 * Register Page — redirects to login since backend handles user provisioning
 * via Stytch magic-link callback. No separate registration step exists.
 */

"use client";

import { useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Loader2, AlertCircle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function RegisterForm() {
    const router = useRouter();
    const searchParams = useSearchParams();
    const sessionRef = searchParams.get("session_ref");

    useEffect(() => {
        // If we have a session_ref, it means Stytch redirected here but
        // the backend callback should have already handled it and set the cookie.
        // Redirect to dashboard to let the proxy validate the session.
        if (sessionRef) {
            router.replace("/");
        } else {
            router.replace("/login");
        }
    }, [router, sessionRef]);

    return (
        <div className="flex min-h-screen items-center justify-center px-4">
            <Card className="w-full max-w-sm">
                <CardHeader className="text-center">
                    <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-yellow-100">
                        <AlertCircle className="h-6 w-6 text-yellow-600" />
                    </div>
                    <CardTitle className="text-2xl">Redirecting…</CardTitle>
                </CardHeader>
                <CardContent className="text-center">
                    <p className="text-muted-foreground mb-4">
                        {sessionRef
                            ? "Completing sign-in…"
                            : "No active sign-in session. Redirecting to login…"}
                    </p>
                    <Loader2 className="text-muted-foreground mx-auto h-8 w-8 animate-spin" />
                </CardContent>
            </Card>
        </div>
    );
}
