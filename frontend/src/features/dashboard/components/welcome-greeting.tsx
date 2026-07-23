"use client";

import { useMe } from "@/hooks/use-auth";

export function WelcomeGreeting() {
    const { data: me } = useMe();
    const greeting = me?.full_name ? `, ${me.full_name}` : "";
    return <>{greeting}</>;
}
