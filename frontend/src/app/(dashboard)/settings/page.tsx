import Link from "next/link";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ThemeSwitch } from "@/components/ui/theme-switch";
import { Button } from "@/components/ui/button";

export default function SettingsPage() {
    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-8 p-8">
            <div>
                <h1 className="text-2xl font-semibold">Settings</h1>
                <p className="text-muted-foreground mt-1">Manage your application preferences.</p>
            </div>

            <Card>
                <CardHeader>
                    <CardTitle>School Settings</CardTitle>
                    <CardDescription>
                        Manage school-level configurations and reference data.
                    </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="font-medium">Behavior Categories</p>
                            <p className="text-muted-foreground">
                                Manage incident/behavior categories for your school.
                            </p>
                        </div>
                        <Button variant="outline" size="sm" asChild>
                            <Link href="/settings/behavior-categories">Manage</Link>
                        </Button>
                    </div>
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="font-medium">Grading Scales</p>
                            <p className="text-muted-foreground">
                                Manage grading scale profiles and percentage-to-level mappings.
                            </p>
                        </div>
                        <Button variant="outline" size="sm" asChild>
                            <Link href="/settings/grading-scales">Manage</Link>
                        </Button>
                    </div>
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="font-medium">Streams</p>
                            <p className="text-muted-foreground">
                                Manage the streams (sections) available in your school.
                            </p>
                        </div>
                        <Button variant="outline" size="sm" asChild>
                            <Link href="/streams/add">Manage</Link>
                        </Button>
                    </div>
                </CardContent>
            </Card>

            <Card>
                <CardHeader>
                    <CardTitle>Appearance</CardTitle>
                    <CardDescription>
                        Choose between light, dark, or system theme mode.
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <div className="flex items-center justify-between">
                        <span className="font-medium">Theme</span>
                        <ThemeSwitch />
                    </div>
                </CardContent>
            </Card>
        </div>
    );
}
