/**
 * Register Page — redirects to login since backend handles user provisioning
 * via Stytch magic-link callback. No separate registration step exists.
 */

import { redirect } from "next/navigation";

export function RegisterForm() {
    // Backend handles user provisioning via Stytch; no registration UI.
    // Redirect to login for all requests.
    redirect("/login");
}
