"use client";

import { Loader2 } from "lucide-react";

export default function LogoutPage() {
    return (
        <div className="flex min-h-screen items-center justify-center">
            <div className="flex flex-col items-center gap-4">
                <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
                <p className="text-muted-foreground">Logging out...</p>
            </div>
        </div>
    );
}
