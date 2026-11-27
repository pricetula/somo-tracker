"use client";

import { Button } from "@/components/ui/button";

interface DataTableErrorProps {
  error: Error;
  retry: () => void;
}

export function DataTableError({ error, retry }: DataTableErrorProps) {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center space-y-3">
      <p className="text-sm text-destructive">Failed to load records.</p>
      <p className="text-xs text-muted-foreground max-w-xs">
        {error.message}
      </p>
      <Button variant="outline" onClick={() => retry()}>
        Retry
      </Button>
    </div>
  );
}
