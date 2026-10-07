"use client";

import { ThemeSwitch } from "@/components/ui/theme-switch";

export default function SettingsPage() {
    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-semibold">Settings</h1>

            <section className="space-y-3">
                <h2 className="text-lg font-medium">Appearance</h2>
                <p className="text-muted-foreground">
                    Switch between light, dark, or system theme.
                </p>
                <ThemeSwitch />
            </section>
        </div>
    );
}
