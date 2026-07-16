/**
 * StudentHealthView — composite view of a student's health profile and incidents.
 *
 * Shows the health profile (blood group, allergies, conditions) and lists
 * recent medical incidents. Allows NURSE/SCHOOL_ADMIN to update the profile.
 */
"use client";

import { useState } from "react";
import { useStudentHealth, useUpsertHealthProfile } from "../hooks/use-health";
import { IncidentList } from "./incident-list";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Pencil } from "lucide-react";
import { toast } from "sonner";

interface StudentHealthViewProps {
    studentId: string;
}

export function StudentHealthView({ studentId }: StudentHealthViewProps) {
    const { data, isLoading } = useStudentHealth(studentId);
    const upsertMutation = useUpsertHealthProfile(studentId);
    const [editing, setEditing] = useState(false);

    const [bloodGroup, setBloodGroup] = useState("");
    const [allergies, setAllergies] = useState("");
    const [chronicConditions, setChronicConditions] = useState("");
    const [emergencyInstructions, setEmergencyInstructions] = useState("");

    const profile = data?.profile;

    // Pre-populate form when profile loads
    const startEditing = () => {
        setBloodGroup(profile?.blood_group ?? "");
        setAllergies((profile?.allergies ?? []).join(", "));
        setChronicConditions((profile?.chronic_conditions ?? []).join(", "));
        setEmergencyInstructions(profile?.emergency_instructions ?? "");
        setEditing(true);
    };

    const handleSave = async () => {
        try {
            await upsertMutation.mutateAsync({
                blood_group: bloodGroup || null,
                allergies: allergies
                    ? allergies
                          .split(",")
                          .map((s) => s.trim())
                          .filter(Boolean)
                    : [],
                chronic_conditions: chronicConditions
                    ? chronicConditions
                          .split(",")
                          .map((s) => s.trim())
                          .filter(Boolean)
                    : [],
                emergency_instructions: emergencyInstructions || null,
            });
            toast.success("Health profile saved.");
            setEditing(false);
        } catch {
            toast.error("Failed to save health profile.");
        }
    };

    if (isLoading) {
        return <p className="text-muted-foreground">Loading health data…</p>;
    }

    return (
        <div className="space-y-6">
            {/* Health Profile Section */}
            <div className="space-y-3">
                <div className="flex items-center justify-between">
                    <h3 className="text-foreground font-medium">Health Profile</h3>
                    {!editing && (
                        <Button variant="outline" size="sm" onClick={startEditing}>
                            <Pencil className="mr-1 size-3" />
                            {profile ? "Edit" : "Add Profile"}
                        </Button>
                    )}
                </div>

                {!profile && !editing ? (
                    <p className="text-muted-foreground">No health profile yet.</p>
                ) : editing ? (
                    <div className="bg-muted/30 space-y-3 rounded-md p-3">
                        <div className="grid grid-cols-2 gap-3">
                            <div className="space-y-1">
                                <Label>Blood Group</Label>
                                <Input
                                    value={bloodGroup}
                                    onChange={(e) => setBloodGroup(e.target.value)}
                                    placeholder="e.g. O+, A-"
                                />
                            </div>
                        </div>
                        <div className="space-y-1">
                            <Label>Allergies (comma-separated)</Label>
                            <Input
                                value={allergies}
                                onChange={(e) => setAllergies(e.target.value)}
                                placeholder="e.g. Peanuts, Penicillin"
                            />
                        </div>
                        <div className="space-y-1">
                            <Label>Chronic Conditions (comma-separated)</Label>
                            <Input
                                value={chronicConditions}
                                onChange={(e) => setChronicConditions(e.target.value)}
                                placeholder="e.g. Asthma, Diabetes"
                            />
                        </div>
                        <div className="space-y-1">
                            <Label>Emergency Instructions</Label>
                            <Textarea
                                value={emergencyInstructions}
                                onChange={(e) => setEmergencyInstructions(e.target.value)}
                                placeholder="Special instructions for emergencies…"
                                rows={2}
                            />
                        </div>
                        <div className="flex gap-2">
                            <Button
                                size="sm"
                                onClick={handleSave}
                                disabled={upsertMutation.isPending}
                            >
                                {upsertMutation.isPending ? "Saving…" : "Save"}
                            </Button>
                            <Button variant="outline" size="sm" onClick={() => setEditing(false)}>
                                Cancel
                            </Button>
                        </div>
                    </div>
                ) : (
                    <div className="space-y-2">
                        {profile?.blood_group && (
                            <p>
                                <span className="text-muted-foreground">Blood Group:</span>{" "}
                                <Badge variant="secondary">{profile.blood_group}</Badge>
                            </p>
                        )}
                        {profile?.allergies && profile.allergies.length > 0 && (
                            <p>
                                <span className="text-muted-foreground">Allergies:</span>{" "}
                                {profile.allergies.join(", ")}
                            </p>
                        )}
                        {profile?.chronic_conditions && profile.chronic_conditions.length > 0 && (
                            <p>
                                <span className="text-muted-foreground">Chronic Conditions:</span>{" "}
                                {profile.chronic_conditions.join(", ")}
                            </p>
                        )}
                        {profile?.emergency_instructions && (
                            <p>
                                <span className="text-muted-foreground">
                                    Emergency Instructions:
                                </span>{" "}
                                {profile.emergency_instructions}
                            </p>
                        )}
                    </div>
                )}
            </div>

            {/* Incidents Section */}
            <IncidentList studentId={studentId} />
        </div>
    );
}
