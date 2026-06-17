"use client";

import { Moon, Sun, Monitor } from "lucide-react";
import { useTheme } from "next-themes";
import { useSyncExternalStore } from "react";
import { cn } from "@/lib/utils";

const THEME_OPTIONS = [
    { value: "light", label: "Light", icon: Sun },
    { value: "dark", label: "Dark", icon: Moon },
    { value: "system", label: "System", icon: Monitor },
] as const;

/**
 * Returns `true` on the client, `false` on the server.
 * Used to prevent hydration mismatch for theme UI.
 */
function useIsClient(): boolean {
    return useSyncExternalStore(
        () => () => {}, // no subscription needed
        () => true, // client snapshot
        () => false // server snapshot
    );
}

export function ThemeSwitch() {
    const { theme, setTheme } = useTheme();
    const isClient = useIsClient();

    // Server / hydration placeholder — inert buttons, no active state
    if (!isClient) {
        return (
            <div className="flex gap-1">
                {THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
                    <button
                        key={value}
                        disabled
                        className={cn(
                            "inline-flex h-9 w-9 items-center justify-center rounded-md",
                            "text-muted-foreground/50",
                            "cursor-default"
                        )}
                        title={label}
                    >
                        <Icon className="h-4 w-4" />
                        <span className="sr-only">{label}</span>
                    </button>
                ))}
            </div>
        );
    }

    return (
        <div className="flex gap-1" role="radiogroup" aria-label="Theme mode">
            {THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
                <button
                    key={value}
                    onClick={() => setTheme(value)}
                    className={cn(
                        "inline-flex h-9 w-9 items-center justify-center rounded-md transition-colors",
                        "focus-visible:ring-ring focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none",
                        theme === value
                            ? "bg-accent text-accent-foreground"
                            : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                    )}
                    title={label}
                    role="radio"
                    aria-checked={theme === value}
                >
                    <Icon className="h-4 w-4" />
                    <span className="sr-only">{label}</span>
                </button>
            ))}
        </div>
    );
}
