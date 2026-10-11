"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { createAcademicPeriod } from "@/features/academic-period/services/api";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function AcademicPeriodAddModalPage() {
    const router = useRouter();
    const handleOpenChange = (open: boolean) => {
        if (!open) router.back();
    };

    const [step, setStep] = useState(1);
    const [year, setYear] = useState("");
    const [terms, setTerms] = useState([{ name: "", startDate: "", endDate: "" }]);

    const handleNext = () => setStep(2);
    const handleBack = () => setStep(1);
    const addTerm = () => setTerms([...terms, { name: "", startDate: "", endDate: "" }]);
    const updateTerm = (index: number, field: string, value: string) => {
        const updated = [...terms];
        updated[index][field as keyof (typeof updated)[0]] = value;
        setTerms(updated);
    };
    const handleSubmit = async () => {
        try {
            const yearNum = Number(year);
            const validTerms = terms.filter((t) => t.name && t.startDate && t.endDate);
            if (!yearNum || validTerms.length === 0) return;
            await createAcademicPeriod({
                year: yearNum,
                terms: validTerms.map((t) => ({
                    name: t.name,
                    startDate: t.startDate,
                    endDate: t.endDate,
                })),
            });
            router.back();
        } catch {}
    };

    return (
        <Dialog open onOpenChange={handleOpenChange}>
            <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-3xl">
                <DialogHeader>
                    <DialogTitle>Create Academic Period</DialogTitle>
                </DialogHeader>
                <div className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle>Step {step} of 2</CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            {step === 1 && (
                                <div className="space-y-4">
                                    <div className="space-y-2">
                                        <Label>Academic Year</Label>
                                        <Select value={year} onValueChange={setYear}>
                                            <SelectTrigger>
                                                <SelectValue placeholder="Select or create year" />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="2026">2026</SelectItem>
                                                <SelectItem value="2027">2027</SelectItem>
                                                <SelectItem value="new">Create new year</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    {year === "new" && (
                                        <div className="space-y-2">
                                            <Label>New Year</Label>
                                            <Input placeholder="Enter year e.g. 2027" />
                                        </div>
                                    )}
                                    <div className="flex justify-end">
                                        <Button onClick={handleNext}>Next</Button>
                                    </div>
                                </div>
                            )}
                            {step === 2 && (
                                <div className="space-y-4">
                                    <Label>Academic Terms for {year}</Label>
                                    {terms.map((term, idx) => (
                                        <div key={idx} className="grid grid-cols-3 gap-3">
                                            <Input
                                                placeholder="Term name"
                                                value={term.name}
                                                onChange={(e) =>
                                                    updateTerm(idx, "name", e.target.value)
                                                }
                                            />
                                            <Input
                                                type="date"
                                                value={term.startDate}
                                                onChange={(e) =>
                                                    updateTerm(idx, "startDate", e.target.value)
                                                }
                                            />
                                            <Input
                                                type="date"
                                                value={term.endDate}
                                                onChange={(e) =>
                                                    updateTerm(idx, "endDate", e.target.value)
                                                }
                                            />
                                        </div>
                                    ))}
                                    <Button variant="outline" onClick={addTerm}>
                                        Add Term
                                    </Button>
                                    <div className="flex justify-between">
                                        <Button variant="outline" onClick={handleBack}>
                                            Back
                                        </Button>
                                        <Button onClick={handleSubmit}>Create Period</Button>
                                    </div>
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </div>
            </DialogContent>
        </Dialog>
    );
}
