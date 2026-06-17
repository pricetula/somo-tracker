import { RegisterPage } from "@/features/auth";
import { getTooltipContent } from "@/lib/docs";

export default function Register() {
    const authSummary = getTooltipContent("authentication");
    return <RegisterPage tooltipSummary={authSummary} />;
}
