"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

interface StudentDetailProps {
    id: string;
}

interface StudentProfile {
    student_id: string;
    admission_number: string;
    nemis_upn?: string;
    full_name: string;
    date_of_birth: string;
    gender: string;
}

export function StudentDetail({ id }: StudentDetailProps) {
    const [profile, setProfile] = useState<StudentProfile | null>(null);
    const [editing, setEditing] = useState(false);
    // TODO: fetch profile via TanStack Query GET /students/:id
    // TODO: PATCH /students/:id on save

    if (!profile) {
        return (
            <div className="p-6">
                <h1 className="text-2xl font-semibold">Student</h1>
                <p className="mt-2">Student ID: {id}</p>
                <p className="text-muted-foreground mt-1">Loading profile...</p>
            </div>
        );
    }

    return (
        <div className="space-y-6 p-6">
            <div className="flex items-center justify-between">
                <h1 className="text-2xl font-semibold">{profile.full_name}</h1>
                <Button variant="outline" onClick={() => setEditing(!editing)}>
                    {editing ? "Cancel" : "Edit"}
                </Button>
            </div>

            <Tabs defaultValue="overview">
                <TabsList>
                    <TabsTrigger value="overview">Overview</TabsTrigger>
                    <TabsTrigger value="academics">Academics</TabsTrigger>
                    <TabsTrigger value="guardians">Guardians</TabsTrigger>
                    <TabsTrigger value="stats">Stats & KPIs</TabsTrigger>
                </TabsList>
                <TabsContent value="overview">
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
                                        value={profile.nemis_upn || ""}
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setProfile({ ...profile, nemis_upn: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Full Name</Label>
                                    <Input
                                        value={profile.full_name}
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setProfile({ ...profile, full_name: e.target.value })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Date of Birth</Label>
                                    <Input
                                        type="date"
                                        value={profile.date_of_birth}
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setProfile({
                                                ...profile,
                                                date_of_birth: e.target.value,
                                            })
                                        }
                                    />
                                </div>
                                <div>
                                    <Label>Gender</Label>
                                    <Input
                                        value={profile.gender}
                                        disabled={!editing}
                                        onChange={(e) =>
                                            setProfile({ ...profile, gender: e.target.value })
                                        }
                                    />
                                </div>
                            </div>
                            {editing && <Button onClick={() => setEditing(false)}>Save</Button>}
                        </CardContent>
                    </Card>
                </TabsContent>
                <TabsContent value="academics">Academics content</TabsContent>
                <TabsContent value="guardians">Guardians content</TabsContent>
                <TabsContent value="stats">Stats & KPIs content</TabsContent>
            </Tabs>
        </div>
    );
}
