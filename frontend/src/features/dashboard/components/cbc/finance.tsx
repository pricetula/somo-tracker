"use client";

import { CheckCircle, AlertCircle } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import type { FinanceData } from "./types";

function formatKES(amount: number): string {
    return `KES ${amount.toLocaleString("en-KE")}`;
}

export function Finance({ data }: { data: FinanceData }) {
    const collectionPct =
        data.totalInvoiced > 0 ? Math.round((data.totalCollected / data.totalInvoiced) * 100) : 0;

    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">Institutional Finance</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="flex flex-col gap-4">
                    {/* Outstanding balance */}
                    <div>
                        {data.outstandingTotal > 0 ? (
                            <>
                                <span className="text-destructive text-[28px] leading-none font-bold">
                                    {formatKES(data.outstandingTotal)}
                                </span>
                                <p className="text-muted-foreground mt-0.5 text-xs">
                                    across {data.invoiceCount} invoices
                                </p>
                            </>
                        ) : (
                            <div className="flex items-center gap-1 text-emerald-600">
                                <CheckCircle className="size-4" />
                                <span className="text-sm font-medium">All fees collected</span>
                            </div>
                        )}
                    </div>

                    <Separator />

                    {/* Fee collection progress */}
                    {data.totalInvoiced > 0 ? (
                        <div className="flex flex-col gap-1">
                            <div className="flex items-center justify-between text-xs">
                                <span>Fee collection — Term 2</span>
                                <span className="font-medium">{collectionPct}%</span>
                            </div>
                            <Progress value={collectionPct} className="h-3" />
                            <div className="text-muted-foreground flex justify-between text-[10px]">
                                <span>{formatKES(data.totalCollected)} collected</span>
                                <span>{formatKES(data.outstandingTotal)} outstanding</span>
                            </div>
                        </div>
                    ) : (
                        <div className="text-muted-foreground py-2 text-xs">
                            No invoices issued for this term yet
                        </div>
                    )}

                    <Separator />

                    {/* Fee categories checklist */}
                    <div>
                        <span className="mb-2 block text-[12px] font-medium">Mandatory fees</span>
                        <div className="flex flex-col gap-1.5">
                            {data.feeCategories.map((cat) => (
                                <label
                                    key={cat.name}
                                    className={`flex items-center gap-2 text-xs ${!cat.configured ? "text-destructive" : ""}`}
                                >
                                    <Checkbox
                                        checked={cat.configured}
                                        disabled
                                        className="size-3.5"
                                    />
                                    {cat.name}
                                    {!cat.configured && <AlertCircle className="size-3" />}
                                </label>
                            ))}
                        </div>
                    </div>
                </div>
            </CardContent>
        </Card>
    );
}
