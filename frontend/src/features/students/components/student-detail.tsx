"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { type StudentProfile, type UpdateStudentRequest } from "../types/student";
import { useStudent, useUpdateStudent } from "../hooks/use-students";

interface StudentDetailProps {
    id: string;
}

export function StudentDetail({ id }: StudentDetailProps) {
    const [editing, setEditing] = useState(false);
    const [draft, setDraft] = useState<Partial<StudentProfile>>({});

    const { data: profile, isLoading, error } = useStudent(id);

    const updateMutation = useUpdateStudent(id);

    if (isLoading) {
        return (
            <div className="p-6">
                <h1 className="text-2xl font-semibold">Student</h1>
                <p className="mt-2">Student ID: {id}</p>
                <p className="text-muted-foreground mt-1">Loading profile...</p>
            </div>
        );
    }

    if (error || !profile) {
        return (
            <div className="p-6">
                <h1 className="text-2xl font-semibold">Student</h1>
                <p className="mt-2">Student ID: {id}</p>
                <p className="text-destructive mt-1">Failed to load profile</p>
            </div>
        );
    }

    const handleSave = () => {
        const payload: UpdateStudentRequest = {
            full_name: draft.full_name,
            date_of_birth: draft.date_of_birth,
            gender: draft.gender,
            nemis_upn: draft.nemis_upn,
            address: draft.address,
            phone: draft.phone,
            email: draft.email,
            nationality: draft.nationality,
            emergency_contact_name: draft.emergency_contact_name,
            emergency_contact_phone: draft.emergency_contact_phone,
            special_needs: draft.special_needs,
        };
        updateMutation.mutate(payload, { onSuccess: () => setEditing(false) });
    };

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold">{profile.full_name}</h1>
                <Button variant="outline" onClick={() => setEditing(!editing)}>
                    {editing ? "Cancel" : "Edit"}
                </Button>
            </div>

            <div className="space-y-6">
                <section className="space-y-2">
                    <h2 className="text-lg font-semibold">Overview</h2>
                    <Card>
                        <CardHeader>
                            <CardTitle>Personal Information</CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <div className="grid grid-cols-2 gap-4">
                                <div>
                                    <Label>Admission Number</Label>
                                    <Input value={profile.admission_number} disabled />
                                </div>
                                <div>
                                    <Label>NEMIS UPN</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.nemis_upn ?? profile.nemis_upn ?? "")
                                                : (profile.nemis_upn ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, nemis_upn: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Full Name</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.full_name ?? profile.full_name)
                                                : profile.full_name
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, full_name: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Date of Birth</Label>
                                    <Input
                                        type="date"
                                        value={
                                            editing
                                                ? (draft.date_of_birth ?? profile.date_of_birth)
                                                : profile.date_of_birth
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, date_of_birth: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Gender</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.gender ?? profile.gender)
                                                : profile.gender
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, gender: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Address</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.address ?? profile.address ?? "")
                                                : (profile.address ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, address: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Phone</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.phone ?? profile.phone ?? "")
                                                : (profile.phone ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, phone: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Email</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.email ?? profile.email ?? "")
                                                : (profile.email ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, email: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Nationality</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.nationality ?? profile.nationality ?? "")
                                                : (profile.nationality ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, nationality: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Emergency Contact Name</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.emergency_contact_name ??
                                                  profile.emergency_contact_name ??
                                                  "")
                                                : (profile.emergency_contact_name ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({
                                                ...draft,
                                                emergency_contact_name: e.target.value,
                                            })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Emergency Contact Phone</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.emergency_contact_phone ??
                                                  profile.emergency_contact_phone ??
                                                  "")
                                                : (profile.emergency_contact_phone ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({
                                                ...draft,
                                                emergency_contact_phone: e.target.value,
                                            })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Special Needs</Label>
                                    <Input
                                        value={
                                            editing
                                                ? (draft.special_needs ??
                                                  profile.special_needs ??
                                                  "")
                                                : (profile.special_needs ?? "")
                                        }
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setDraft({ ...draft, special_needs: e.target.value })
                                        }
                                    />
                                </div>
                            </div>
                            {editing && (
                                <div className="flex gap-2">
                                    <Button
                                        onClick={handleSave}
                                        disabled={updateMutation.isPending}
                                    >
                                        {updateMutation.isPending ? "Saving..." : "Save"}
                                    </Button>
                                    <Button
                                        variant="outline"
                                        onClick={() => {
                                            setEditing(false);
                                            setDraft({});
                                        }}
                                    >
                                        Cancel
                                    </Button>
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </section>

                <section className="space-y-2">
                    <h2 className="text-lg font-semibold">Academics</h2>
                    <Card>
                        <CardContent>
                            <p className="text-muted-foreground">Academics content</p>
                        </CardContent>
                    </Card>
                </section>

                <section className="space-y-2">
                    <h2 className="text-lg font-semibold">Guardians</h2>
                    <Card>
                        <CardContent className="space-y-4">
                            <p className="text-muted-foreground">
                                Link guardians to this student and manage relationship types.
                            </p>
                            <div className="text-muted-foreground text-sm">
                                Guardian linking UI placeholder – backend API for link/unlink
                                endpoints to be wired.
                            </div>
                        </CardContent>
                    </Card>
                </section>

                <section className="space-y-2">
                    <h2 className="text-lg font-semibold">Stats & KPIs</h2>
                    <Card>
                        <CardContent>
                            <p className="text-muted-foreground">Stats & KPIs content</p>
                        </CardContent>
                    </Card>
                </section>
            </div>
        </div>
    );
}
