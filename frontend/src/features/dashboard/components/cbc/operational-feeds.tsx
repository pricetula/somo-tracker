"use client";

import { useMemo } from "react";
import { HeartPulse } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { PendingUser, MedicalIncident } from "./types";

function getInitials(name: string): string {
    return name
        .split(" ")
        .map((n) => n[0])
        .join("")
        .toUpperCase()
        .slice(0, 2);
}

const MS_PER_WEEK = 7 * 24 * 60 * 60 * 1000;

// ── 2F-1: Pre-invited User Queue ──
function PendingUserQueue({ users }: { users: PendingUser[] }) {
    const maxShow = 5;
    const remaining = users.length - maxShow;

    const enrichedUsers = useMemo(() => {
        const now = new Date().getTime();
        return users.slice(0, maxShow).map((user) => ({
            user,
            invitedDate: new Date(user.invitedAt),
            daysAgo: formatDistanceToNow(new Date(user.invitedAt), { addSuffix: true }),
            isStale: now - new Date(user.invitedAt).getTime() > MS_PER_WEEK,
        }));
    }, [users]);

    if (users.length === 0) {
        return (
            <div>
                <span className="mb-2 block text-[11px] font-medium">Pending Invitations</span>
                <p className="text-muted-foreground text-[11px]">No pending invitations</p>
            </div>
        );
    }

    return (
        <div>
            <span className="mb-2 block text-[11px] font-medium">Pending Invitations</span>
            <div className="flex flex-col gap-2">
                {enrichedUsers.map(({ user, daysAgo, isStale }) => {
                    return (
                        <div key={user.email} className="flex items-center gap-2">
                            <Avatar className="size-6">
                                <AvatarFallback className="text-[9px]">
                                    {getInitials(user.name)}
                                </AvatarFallback>
                            </Avatar>
                            <div className="min-w-0 flex-1">
                                <div className="truncate text-xs font-medium">{user.name}</div>
                                <div className="flex items-center gap-1">
                                    <Badge variant="secondary" className="text-[9px]">
                                        {user.role.replace("_", " ")}
                                    </Badge>
                                    <span
                                        className={`text-[10px] ${isStale ? "text-amber-600" : "text-muted-foreground"}`}
                                    >
                                        {daysAgo}
                                    </span>
                                </div>
                            </div>
                            <Button variant="ghost" size="sm" className="h-6 shrink-0 text-[10px]">
                                Resend
                            </Button>
                        </div>
                    );
                })}
                {remaining > 0 && (
                    <Button variant="link" size="sm" className="h-auto self-start p-0 text-[10px]">
                        View all ({users.length})
                    </Button>
                )}
            </div>
        </div>
    );
}

// ── 2F-2: Recent Medical Incidents ──
const MAX_SYMPTOMS_LENGTH = 60;

function MedicalIncidentList({ incidents }: { incidents: MedicalIncident[] }) {
    if (incidents.length === 0) {
        return (
            <div>
                <span className="mb-2 block text-[11px] font-medium">
                    Medical Incidents — Today
                </span>
                <div className="text-muted-foreground flex items-center gap-2 text-[11px]">
                    <HeartPulse className="size-4" />
                    No medical incidents recorded today
                </div>
            </div>
        );
    }

    return (
        <div>
            <span className="mb-2 block text-[11px] font-medium">Medical Incidents — Today</span>
            <div className="flex flex-col">
                {incidents.map((inc, i) => {
                    const truncated = inc.symptoms.length > MAX_SYMPTOMS_LENGTH;
                    const displaySymptoms = truncated
                        ? inc.symptoms.slice(0, MAX_SYMPTOMS_LENGTH) + "..."
                        : inc.symptoms;

                    return (
                        <div key={i} className="relative flex gap-3 pb-3 last:pb-0">
                            {/* Timeline line */}
                            <div className="flex flex-col items-center">
                                <div className="bg-muted-foreground size-2 rounded-full" />
                                {i < incidents.length - 1 && (
                                    <div className="bg-border mt-1 w-[2px] flex-1" />
                                )}
                            </div>
                            <div className="flex min-w-0 flex-1 flex-col gap-0.5 text-xs">
                                <div className="flex items-center gap-1.5">
                                    <span className="text-muted-foreground font-mono text-[11px]">
                                        {inc.time}
                                    </span>
                                    <span className="font-medium">{inc.student}</span>
                                    <span className="bg-muted text-muted-foreground rounded px-1 py-0.5 text-[10px]">
                                        {inc.class}
                                    </span>
                                </div>
                                <div className="text-muted-foreground text-[13px]">
                                    {truncated ? (
                                        <Tooltip>
                                            <TooltipTrigger asChild>
                                                <span className="cursor-help">
                                                    {displaySymptoms}
                                                </span>
                                            </TooltipTrigger>
                                            <TooltipContent
                                                side="top"
                                                className="max-w-[260px] text-xs"
                                            >
                                                {inc.symptoms}
                                            </TooltipContent>
                                        </Tooltip>
                                    ) : (
                                        displaySymptoms
                                    )}
                                </div>
                                <div className="text-[13px]">{inc.action}</div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
}

// ── Combined Driver ──
export function OperationalFeeds({
    pendingUsers,
    medicalIncidents,
}: {
    pendingUsers: PendingUser[];
    medicalIncidents: MedicalIncident[];
}) {
    return (
        <Card>
            <CardHeader>
                <CardTitle className="text-xs">Real-time Operational Feeds</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="flex flex-col gap-3">
                    <PendingUserQueue users={pendingUsers} />
                    <Separator />
                    <MedicalIncidentList incidents={medicalIncidents} />
                </div>
            </CardContent>
        </Card>
    );
}
