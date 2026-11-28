"use client";

import { cn } from "@/components/shared/data-table/utils";

export function DataTableLoading() {
  return (
    <div className="flex flex-col items-center justify-center py-12">
      <p className="text-muted-foreground">Loading records…</p>
    </div>
  );
}
