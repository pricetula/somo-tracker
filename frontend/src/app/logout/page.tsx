"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
// import { toast } from "sonner";
import { Loader2 } from "lucide-react";
// import { logout } from "@/lib/api/auth";
// import { SESSION_COOKIE_NAME, ROLE_COOKIE_NAME } from "@/lib/auth";
// import { getErrorMessage } from "@/lib/errors";

export default function LogoutPage() {
    const router = useRouter();
    const queryClient = useQueryClient();

    useEffect(() => {
        async function doLogout() {
            // try {
            //     await logout();
            //     queryClient.clear();
            //     toast.success("Logged out");
            // } catch (err) {
            //     // Session may already be expired or backend unreachable —
            //     // still redirect to /login. When the API call fails (network
            //     // error, backend down), cookies are NOT cleared server-side,
            //     // so we clear them here to prevent the proxy middleware from
            //     // seeing stale cookies and bouncing back from /login to /.
            //     console.warn("logout: session deletion failed", getErrorMessage(err));
            //     document.cookie = `${SESSION_COOKIE_NAME}=; path=/; max-age=0`;
            //     document.cookie = `${ROLE_COOKIE_NAME}=; path=/; max-age=0`;
            //     document.cookie = "csrf_token=; path=/; max-age=0";
            // } finally {
            //     router.replace("/login");
            // }
        }

        doLogout();
    }, [router, queryClient]);

    return (
        <div className="flex min-h-screen items-center justify-center">
            <div className="flex flex-col items-center gap-4">
                <Loader2 className="text-muted-foreground h-8 w-8 animate-spin" />
                <p className="text-muted-foreground">Logging out...</p>
            </div>
        </div>
    );
}
