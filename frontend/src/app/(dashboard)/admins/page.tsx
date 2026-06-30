/**
 * Admins listing page — active school administrators.
 *
 * Uses its own query hook and table component — not the generic
 * members module. Maps to GET /api/v1/members?role=SCHOOL_ADMIN.
 *
 * Invitations are listed on the dedicated /admins/invitations page.
 */

"use client";

import * as React from "react";
import Link from "next/link";

import { AdminsTable } from "@/features/staff/components/admins-table";
import { useAdmins } from "@/features/staff/hooks/use-admins";
import { Button } from "@/components/ui/button";
import { Send } from "lucide-react";

export default function AdminsPage() {
    const {
        data: adminsData,
        isLoading: adminsLoading,
        isError: adminsError,
    } = useAdmins({ includeInactive: true });

    return (
        <div className="flex flex-1 flex-col">
            {/* Page header */}
            <div className="flex items-center gap-3 px-6 pt-6 pb-2">
                <h1 className="text-2xl font-semibold tracking-tight">Admins</h1>
                <div className="ml-auto">
                    <Button variant="outline" size="sm" asChild>
                        <Link href="/admins/invitations">
                            <Send className="mr-1.5 size-3.5" />
                            Invitations
                        </Link>
                    </Button>
                </div>
            </div>

            <div className="flex flex-1 flex-col px-6 py-4">
                <section className="flex flex-1 flex-col">
                    {adminsError ? (
                        <div className="flex items-center justify-center py-8">
                            <p className="text-destructive text-sm">
                                Failed to load admins. Please try again.
                            </p>
                        </div>
                    ) : (
                        <div className="ring-foreground/10 rounded-lg ring-1">
                            <AdminsTable
                                admins={adminsData?.members ?? []}
                                total={adminsData?.total ?? 0}
                                isLoading={adminsLoading}
                            />
                        </div>
                    )}
                </section>
            </div>
        </div>
    );
}
