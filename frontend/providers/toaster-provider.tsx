"use client";

import { Toaster } from "sonner";
import { type ReactNode } from "react";

export function ToasterProvider({ children }: { children: ReactNode }) {
  return (
    <>
      {children}
      <Toaster
        position="top-right"
        toastOptions={{
          classNames: {
            toast: "bg-background text-foreground border-border",
            description: "text-muted-foreground",
            actionButton: "bg-primary text-primary-foreground",
            cancelButton: "bg-secondary text-secondary-foreground",
          },
        }}
      />
    </>
  );
}