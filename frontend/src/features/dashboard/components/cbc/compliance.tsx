"use client";

import { AlertCircle, AlertTriangle, CheckCircle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import type { ComplianceItem } from "./types";

export function Compliance({ items }: { items: ComplianceItem[] }) {
    const visible = items.filter((i) => i.count > 0);

    if (visible.length === 0) {
        return (
            <Card>
                <CardHeader>
                    <CardTitle className="text-xs">System Compliance</CardTitle>
                </CardHeader>
                <CardContent>
                    <Alert className="border-emerald-300 bg-emerald-50 text-emerald-800">
                        <CheckCircle className="size-4" />
                        <AlertDescription>All compliance checks passed</AlertDescription>
                    </Alert>
                </CardContent>
            </Card>
        );
    }

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">System Compliance</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="flex flex-col gap-2">
                    {visible.map((item) => {
                        const isError = item.severity === "error";
                        return (
                            <Alert
                                key={item.id}
                                variant={isError ? "destructive" : "default"}
                                className={!isError ? "border-amber-300" : ""}
                            >
                                <div className="flex w-full items-center gap-2">
                                    {isError ? (
                                        <AlertCircle className="size-4 shrink-0" />
                                    ) : (
                                        <AlertTriangle className="size-4 shrink-0 text-amber-600" />
                                    )}
                                    <AlertDescription className="flex w-full items-center justify-between gap-2 text-xs">
                                        <span>
                                            {item.issue}{" "}
                                            <span className="font-bold">{item.count}</span>
                                        </span>
                                        <Button
                                            variant="link"
                                            size="sm"
                                            className="h-auto shrink-0 p-0 text-xs"
                                        >
                                            {item.resolveLabel}
                                        </Button>
                                    </AlertDescription>
                                </div>
                            </Alert>
                        );
                    })}
                </div>
            </CardContent>
        </Card>
    );
}
