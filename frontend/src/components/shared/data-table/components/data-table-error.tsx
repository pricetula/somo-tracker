"use client";

import { Button } from "@/components/ui/button";
import { getErrorMessage } from "../utils";

interface DataTableErrorProps {
  error: unknown;
  retry: () => void;
}

export function DataTableError({ error, retry }: DataTableErrorProps) {
  const message = getErrorMessage(error);
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center space-y-3">
      <p className="text-destructive">Failed to load records.</p>
      <p className="text-muted-foreground max-w-xs">{message}</p>
      <Button variant="outline" onClick={() => retry()}>
        Retry
      </Button>
    </div>
  );
}
