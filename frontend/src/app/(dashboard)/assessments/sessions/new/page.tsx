"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
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

export default function CreateAssessmentPage() {
    const router = useRouter();
    const [name, setName] = useState("");
    const [method, setMethod] = useState<"QUANTITATIVE" | "RUBRIC">("QUANTITATIVE");
    const [maxPoints, setMaxPoints] = useState(100);
    const [profileId, setProfileId] = useState("");
    const [profiles, setProfiles] = useState<{ id: string; name: string }[]>([]);
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        fetch("/api/v1/assessments/grading-scale-profiles")
            .then((r) => r.json())
            .then((data) => setProfiles(data.items ?? []))
            .catch(() => setProfiles([]));
    }, []);

    async function handleSubmit(e: React.FormEvent) {
        e.preventDefault();
        setLoading(true);
        try {
            const payload: Record<string, unknown> = {
                name,
                evaluation_method: method,
                class_id: "placeholder",
                learning_area_id: "placeholder",
            };
            if (method === "QUANTITATIVE") {
                payload.max_points = maxPoints;
                payload.grading_scale_profile_id = profileId || undefined;
            }
            await fetch("/api/v1/assessments/sessions", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            });
            router.push("/assessments/sessions");
        } catch {
            setLoading(false);
        }
    }

    return (
        <div className="mx-auto max-w-md p-6">
            <h1 className="mb-6 text-xl font-semibold">New Assessment Session</h1>
            <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                    <Label htmlFor="name">Name</Label>
                    <Input
                        id="name"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        required
                    />
                </div>
                <div>
                    <Label htmlFor="method">Method</Label>
                    <Select
                        value={method}
                        onValueChange={(v) => setMethod(v as "QUANTITATIVE" | "RUBRIC")}
                    >
                        <SelectTrigger id="method">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="QUANTITATIVE">Quantitative</SelectItem>
                            <SelectItem value="RUBRIC">Rubric</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
                {method === "QUANTITATIVE" && (
                    <>
                        <div>
                            <Label htmlFor="max">Max Points</Label>
                            <Input
                                id="max"
                                type="number"
                                value={maxPoints}
                                onChange={(e) => setMaxPoints(Number(e.target.value))}
                            />
                        </div>
                        <div>
                            <Label htmlFor="profile">Grading Scale Profile</Label>
                            <Select value={profileId} onValueChange={setProfileId}>
                                <SelectTrigger id="profile">
                                    <SelectValue placeholder="Select a profile" />
                                </SelectTrigger>
                                <SelectContent>
                                    {profiles.map((p) => (
                                        <SelectItem key={p.id} value={p.id}>
                                            {p.name}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                    </>
                )}
                <Button type="submit" disabled={loading} className="w-full">
                    {loading ? "Creating..." : "Create Session"}
                </Button>
            </form>
        </div>
    );
}
