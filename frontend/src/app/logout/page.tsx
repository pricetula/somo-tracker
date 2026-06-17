"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Loader2 } from "lucide-react";
import { logout } from "@/lib/api/auth";

export default function LogoutPage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    useEffect(() => {
        async function doLogout() {
            try {
                await logout();
                queryClient.clear();
                toast.success("Logged out");
            } catch {
                // Session may already be expired — still redirect
            } finally {
                router.replace("/login");
            }
        }

        doLogout();
    }, [router, queryClient]);

    return (
        <div className="flex min-h-screen items-center justify-center">
            <div className="flex flex-col items-center gap-4">
                <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
                <p className="text-muted-foreground text-sm">Logging out...</p>
            </div>
        </div>
    );
}
