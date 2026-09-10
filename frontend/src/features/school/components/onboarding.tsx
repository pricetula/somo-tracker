/**
 * OnboardingForm — school registration onboarding.
 *
 * Uses shadcn Form + zod + react-hook-form. Fields: school_name, user_name.
 * Integrates with useRegisterSchool mutation.
 */

"use client";

import React from "react";
import { OnboardingForm } from "./onboarding-form";
import { CreateAcademicYear } from "./create-academic-year";
import { CreateStreams } from "./create-streams";
import { useRouter } from "next/navigation";

export function Onboarding() {
    const router = useRouter();
    const [stage, setStage] = React.useState(0);

    return (
        <>
            {(stage === 0 && (
                <OnboardingForm
                    onSuccess={(s) => {
                        if (s) {
                            setStage(1);
                        }
                    }}
                />
            )) ||
                (stage === 1 && <CreateAcademicYear onSuccess={() => setStage(1)} />) ||
                (stage === 2 && <CreateStreams onSuccess={() => router.push("/")} />)}
        </>
    );
}
