"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api/client";
import type { MeResult } from "@/features/auth/lib/types";
import type { ReactNode } from "react";

export function DashboardAuthLayout({ children }: { children: ReactNode }) {
    const {
        data: me,
        isLoading,
        isError,
    } = useQuery<MeResult, Error>({
        queryKey: ["me"],
        queryFn: async () => {
            const res = await api.get<MeResult>("/api/me");
            return res;
        },
        staleTime: 60_000,
        retry: 1,
        refetchOnWindowFocus: false,
    });

    if (isLoading) {
        return <div>Loading session…</div>;
    }

    if (isError || !me) {
        return <div>Authentication error.</div>;
    }

    // Auth guard: active school context
    // if (!me.active_school_id || !me.active_school_role) {
    //   window.location.href = "/select-school";
    //   return null;
    // }

    return <>{children}</>;
}
