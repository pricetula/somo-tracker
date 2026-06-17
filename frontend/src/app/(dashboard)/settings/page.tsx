import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ThemeSwitch } from "@/components/ui/theme-switch";

export default function SettingsPage() {
    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-8 p-8">
            <div>
                <h1 className="text-2xl font-semibold">Settings</h1>
                <p className="text-muted-foreground mt-1 text-sm">
                    Manage your application preferences.
                </p>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>Appearance</CardTitle>
                    <CardDescription>
                        Choose between light, dark, or system theme mode.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="flex items-center justify-between">
                        <span className="text-sm font-medium">Theme</span>
                        <ThemeSwitch />
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
