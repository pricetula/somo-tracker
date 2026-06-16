import { LoginPage } from "@/features/auth";
import { getTooltipContent } from "@/lib/docs";

export default function Login() {
  const authSummary = getTooltipContent("authentication");
  return <LoginPage tooltipSummary={authSummary} />;
}
