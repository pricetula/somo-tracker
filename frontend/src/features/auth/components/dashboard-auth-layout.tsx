import type { ReactNode } from "react";
import { redirect } from "next/navigation";
import { serverApi } from "@/lib/api/server";
import type { MeResult } from "@/features/auth/lib/types";

interface CustomError {
    status: number;
    code: string;
    errors: object;
    name: string;
}

export async function DashboardAuthLayout({ children }: { children: ReactNode }) {
    let me: MeResult | null = null;

    try {
        me = await serverApi.get<MeResult>("/api/me");
    } catch (e) {
        const err = e as CustomError;
        if (err && err.status && err.status === 401) {
            // return redirect("/login");
        }
    }

    if (!me) return redirect("/logout");

    if (!me.active_school_id) return redirect("/register");

    // Auth guard: active school context
    // if (!me.active_school_id || !me.active_school_role) {
    //   window.location.href = "/select-school";
    //   return null;
    // }

    return <>{children}</>;
}
