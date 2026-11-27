"use client";

export function DataTableEmpty() {
  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <p className="text-xs text-muted-foreground">No records found.</p>
    </div>
  );
}
