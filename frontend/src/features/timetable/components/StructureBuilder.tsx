"use client";

import { useState } from "react";
import { Plus, Minus, Copy, Check, X, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Card,
    CardContent,
    CardFooter,
    CardHeader,
    CardTitle,
    CardDescription,
} from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
    DAYS_OF_WEEK,
    type DayOfWeek,
    type PeriodFormData,
    type StructureBuilderStep,
} from "../types";

const DEFAULT_PERIODS: Omit<PeriodFormData, "dayOfWeek">[] = [
    { periodName: "Period 1", startTime: "08:00", endTime: "08:40", isBreak: false },
    { periodName: "Period 2", startTime: "08:45", endTime: "09:25", isBreak: false },
    { periodName: "Break", startTime: "09:25", endTime: "09:45", isBreak: true },
    { periodName: "Period 3", startTime: "09:45", endTime: "10:25", isBreak: false },
    { periodName: "Period 4", startTime: "10:30", endTime: "11:10", isBreak: false },
    { periodName: "Lunch", startTime: "11:10", endTime: "12:00", isBreak: true },
    { periodName: "Period 5", startTime: "12:00", endTime: "12:40", isBreak: false },
    { periodName: "Period 6", startTime: "12:45", endTime: "13:25", isBreak: false },
];

interface StructureBuilderProps {
    onComplete: (periods: PeriodFormData[]) => void;
    onCancel: () => void;
    initialPeriods?: PeriodFormData[];
}

export function StructureBuilder({ onComplete, onCancel, initialPeriods }: StructureBuilderProps) {
    const [step, setStep] = useState<StructureBuilderStep>("periods");
    const [periods, setPeriods] = useState<PeriodFormData[]>(
        initialPeriods?.length
            ? initialPeriods
            : DEFAULT_PERIODS.map((p) => ({ ...p, dayOfWeek: 1 }))
    );
    const [replicateSourceDay, setReplicateSourceDay] = useState<DayOfWeek | null>(1);
    const [replicateTargetDays, setReplicateTargetDays] = useState<DayOfWeek[]>([2, 3, 4, 5, 6]);

    const addPeriod = () => {
        const lastPeriod = periods[periods.length - 1];
        const [lastHours, lastMinutes] = lastPeriod.endTime.split(":").map(Number);
        const startMinutes = lastHours * 60 + lastMinutes + 5;
        const startHours = Math.floor(startMinutes / 60);
        const startMins = startMinutes % 60;
        const newStart = `${String(startHours).padStart(2, "0")}:${String(startMins).padStart(2, "0")}`;
        const endMinutes = startMinutes + 40;
        const endHours = Math.floor(endMinutes / 60);
        const endMins = endMinutes % 60;
        const newEnd = `${String(endHours).padStart(2, "0")}:${String(endMins).padStart(2, "0")}`;

        setPeriods([
            ...periods,
            {
                dayOfWeek: 1,
                periodName: `Period ${periods.length + 1}`,
                startTime: newStart,
                endTime: newEnd,
                isBreak: false,
            },
        ]);
    };

    const removePeriod = (index: number) => {
        setPeriods(periods.filter((_, idx) => idx !== index));
    };

    const updatePeriod = (index: number, field: keyof PeriodFormData, value: string | boolean) => {
        setPeriods(periods.map((p, i) => (i === index ? { ...p, [field]: value } : p)));
    };

    const handleReplicate = () => {
        const sourcePeriods = periods.filter((p) => p.dayOfWeek === replicateSourceDay);
        const newPeriods = [...periods];
        for (const targetDay of replicateTargetDays) {
            if (targetDay === replicateSourceDay) continue;
            for (const p of sourcePeriods) {
                newPeriods.push({ ...p, dayOfWeek: targetDay });
            }
        }
        setPeriods(newPeriods);
    };

    const handleComplete = () => {
        // Sort by day, then by start_time
        const sorted = [...periods].sort((a, b) => {
            if (a.dayOfWeek !== b.dayOfWeek) return a.dayOfWeek - b.dayOfWeek;
            return a.startTime.localeCompare(b.startTime);
        });
        onComplete(sorted);
    };

    const renderPeriodsStep = () => (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h4 className="font-medium">Monday (Template Day)</h4>
                <Button variant="outline" size="sm" onClick={addPeriod}>
                    <Plus className="mr-1 h-4 w-4" />
                    Add Period
                </Button>
            </div>

            <div className="max-h-96 space-y-3 overflow-y-auto">
                {periods
                    .filter((p) => p.dayOfWeek === 1)
                    .map((period, index) => (
                        <div
                            key={index}
                            className="bg-background flex items-center gap-2 rounded-md border p-3"
                        >
                            <Input
                                placeholder="Period name"
                                value={period.periodName}
                                onChange={(e) => updatePeriod(index, "periodName", e.target.value)}
                                className="w-28"
                            />
                            <Input
                                type="time"
                                value={period.startTime}
                                onChange={(e) => updatePeriod(index, "startTime", e.target.value)}
                                className="w-24"
                            />
                            <span className="text-muted-foreground">–</span>
                            <Input
                                type="time"
                                value={period.endTime}
                                onChange={(e) => updatePeriod(index, "endTime", e.target.value)}
                                className="w-24"
                            />
                            <div className="ml-auto flex items-center gap-2">
                                <Label className="cursor-pointer text-sm">
                                    <input
                                        type="checkbox"
                                        checked={period.isBreak}
                                        onChange={(e) =>
                                            updatePeriod(index, "isBreak", e.target.checked)
                                        }
                                        className="peer sr-only"
                                    />
                                    <span className="bg-muted peer-focus:ring-ring peer-checked:bg-primary after:bg-background relative flex h-5 w-9 items-center rounded-full peer-focus:ring-2 peer-focus:ring-offset-2 peer-focus:outline-none after:absolute after:top-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:transition-all peer-checked:after:translate-x-4"></span>
                                    Break
                                </Label>
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => removePeriod(index)}
                                >
                                    <Minus className="h-4 w-4" />
                                </Button>
                            </div>
                        </div>
                    ))}
            </div>

            <Separator />

            <div className="space-y-3">
                <h4 className="font-medium">Replicate to Other Days</h4>
                <div className="flex items-center gap-4">
                    <div className="flex-1">
                        <Label htmlFor="source-day">Copy from</Label>
                        <Select value={replicateSourceDay} onValueChange={setReplicateSourceDay}>
                            <SelectTrigger id="source-day">
                                <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                                {DAYS_OF_WEEK.map((d) => (
                                    <SelectItem key={d.value} value={d.value}>
                                        {d.label}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                    </div>
                    <div className="flex-1">
                        <Label>To days</Label>
                        <div className="flex flex-wrap gap-2">
                            {DAYS_OF_WEEK.filter((d) => d.value !== replicateSourceDay).map((d) => (
                                <label
                                    key={d.value}
                                    className="hover:bg-accent inline-flex cursor-pointer items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm transition-colors"
                                >
                                    <input
                                        type="checkbox"
                                        checked={replicateTargetDays.includes(d.value)}
                                        onChange={(e) =>
                                            setReplicateTargetDays(
                                                e.target.checked
                                                    ? [...replicateTargetDays, d.value]
                                                    : replicateTargetDays.filter(
                                                          (td) => td !== d.value
                                                      )
                                            )
                                        }
                                        className="text-primary focus:ring-primary h-4 w-4 rounded border-gray-300"
                                    />
                                    {d.short}
                                </label>
                            ))}
                        </div>
                    </div>
                </div>
                <Button
                    variant="outline"
                    onClick={handleReplicate}
                    disabled={replicateTargetDays.length === 0}
                >
                    <Copy className="mr-2 h-4 w-4" />
                    Replicate Schedule
                </Button>
            </div>
        </div>
    );

    const renderReviewStep = () => (
        <div className="max-h-96 space-y-4 overflow-y-auto">
            <p className="text-muted-foreground text-sm">
                Review the weekly schedule. Periods are grouped by day.
            </p>
            {DAYS_OF_WEEK.map(({ value: day, label }) => {
                const dayPeriods = periods.filter((p) => p.dayOfWeek === day);
                if (dayPeriods.length === 0) return null;
                return (
                    <div key={day} className="rounded-md border p-3">
                        <div className="mb-2 font-medium">{label}</div>
                        <div className="space-y-1 text-sm">
                            {dayPeriods.map((p, i) => (
                                <div
                                    key={i}
                                    className="text-muted-foreground flex items-center justify-between"
                                >
                                    <span>{p.periodName}</span>
                                    <span>
                                        {p.startTime} – {p.endTime} {p.isBreak && "(Break)"}
                                    </span>
                                </div>
                            ))}
                        </div>
                    </div>
                );
            })}
        </div>
    );

    return (
        <Card className="w-full max-w-2xl">
            <CardHeader>
                <CardTitle>Create Timetable Structure</CardTitle>
                <CardDescription>
                    Define the weekly schedule template. Start with Monday, then replicate to other
                    days.
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                {step === "periods" && renderPeriodsStep()}
                {step === "review" && renderReviewStep()}
            </CardContent>
            <CardFooter className="flex justify-between">
                <Button variant="ghost" onClick={onCancel}>
                    <X className="mr-2 h-4 w-4" />
                    Cancel
                </Button>
                <div className="flex gap-2">
                    {step === "periods" && (
                        <Button onClick={() => setStep("review")}>
                            Review <ChevronRight className="ml-2 h-4 w-4" />
                        </Button>
                    )}
                    {step === "review" && (
                        <>
                            <Button variant="outline" onClick={() => setStep("periods")}>
                                <ChevronLeft className="mr-2 h-4 w-4" />
                                Back
                            </Button>
                            <Button onClick={handleComplete}>
                                <Check className="mr-2 h-4 w-4" />
                                Create Structure
                            </Button>
                        </>
                    )}
                </div>
            </CardFooter>
        </Card>
    );
}
