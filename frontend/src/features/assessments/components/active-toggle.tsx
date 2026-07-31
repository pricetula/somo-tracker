"use client";

import { ToggleLeft, ToggleRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { type ScaleProfile } from "@/lib/api/assessments";
import { useToggleScaleProfile } from "../hooks/use-assessments";

export function ActiveToggle({ profile }: { profile: ScaleProfile }) {
    const toggleMutation = useToggleScaleProfile();

    return (
        <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => toggleMutation.mutate({ id: profile.id, isActive: !profile.is_active })}
            disabled={toggleMutation.isPending}
            title={profile.is_active ? "Deactivate" : "Activate"}
        >
            {profile.is_active ? (
                <ToggleRight className="h-4 w-4 text-emerald-600" />
            ) : (
                <ToggleLeft className="text-muted-foreground h-4 w-4" />
            )}
        </Button>
    );
}
