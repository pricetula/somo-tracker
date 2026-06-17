"use client";

import { ShieldAlert, CheckCircle } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import type { HealthFlagsData } from "./types";

export function HealthFlags({ data }: { data: HealthFlagsData }) {
    if (data.count === 0) {
        return (
            <Card className="border-emerald-500">
                <CardContent>
                    <div className="flex items-center gap-2 py-1">
                        <CheckCircle className="size-5 text-emerald-600" />
                        <span className="text-muted-foreground text-xs">
                            No critical health flags today
                        </span>
                    </div>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card className="border-destructive">
            <CardContent>
                <div className="flex flex-col gap-3">
                    <div className="flex items-center gap-2">
                        <ShieldAlert className="text-destructive size-6" />
                        <div>
                            <span className="text-destructive text-[32px] leading-none font-bold">
                                {data.count}
                            </span>
                            <span className="text-muted-foreground ml-1 text-xs">
                                students with critical health flags
                            </span>
                        </div>
                    </div>

                    <Sheet>
                        <SheetTrigger asChild>
                            <Button variant="outline" size="sm" className="w-full">
                                View details
                            </Button>
                        </SheetTrigger>
                        <SheetContent
                            side="right"
                            className="w-[400px] overflow-y-auto sm:w-[440px]"
                        >
                            <SheetHeader>
                                <SheetTitle>Critical Health Flags</SheetTitle>
                            </SheetHeader>
                            <div className="mt-4 flex flex-col gap-4">
                                {data.students.map((s) => (
                                    <div key={s.name} className="rounded-lg border p-3">
                                        <div className="mb-1 text-sm font-medium">{s.name}</div>
                                        <div className="text-muted-foreground mb-2 text-xs">
                                            {s.class}
                                        </div>
                                        <div className="mb-2 flex flex-wrap gap-1">
                                            {s.conditions.map((c) => (
                                                <Badge
                                                    key={c}
                                                    variant="destructive"
                                                    className="text-[10px]"
                                                >
                                                    {c}
                                                </Badge>
                                            ))}
                                        </div>
                                        <pre className="bg-muted text-muted-foreground rounded px-2 py-1.5 text-[11px] whitespace-pre-wrap">
                                            {s.emergency}
                                        </pre>
                                    </div>
                                ))}
                            </div>
                        </SheetContent>
                    </Sheet>
                </div>
            </CardContent>
        </Card>
    );
}
