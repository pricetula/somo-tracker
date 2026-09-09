/**
 * OnboardingForm — school registration onboarding.
 *
 * Uses shadcn Form + zod + react-hook-form. Fields: school_name, user_name.
 * Integrates with useRegisterSchool mutation.
 */

"use client";

import React from "react";
import { OnboardingForm } from "./onboarding-form";

export function Onboarding() {
    const [schoolId, setSchoolId] = React.useState("");

    if (schoolId) {
    }

    return (
        <>
            <OnboardingForm onSuccess={setSchoolId} />
        </>
    );
}
