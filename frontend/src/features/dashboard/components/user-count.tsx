"use client";

import React from "react";
import Link from "next/link";
import { numberCompactor } from "@/lib/number-compactor";

export function UserCount() {
    return (
        <section>
            <Link href="/students" className="flex flex-col items-center no-underline!">
                <span className="text-2xl no-underline!">{numberCompactor(2239)}</span>
                <span className="text-muted-foreground text-xs no-underline!">Students</span>
            </Link>
        </section>
    );
}
