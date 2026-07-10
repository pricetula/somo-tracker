import { WelcomeGreeting } from "./welcome-greeting";

export function TeacherDashboardPage() {
    return (
        <div className="flex flex-1 flex-col gap-6 p-6">
            <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                    Welcome back
                    <WelcomeGreeting />
                </h1>
                <p className="text-muted-foreground mt-1">Welcome to SomoTracker.</p>
            </div>
        </div>
    );
}
