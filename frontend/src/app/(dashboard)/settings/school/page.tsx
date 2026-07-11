import { StreamsSection } from "@/features/settings-school";

export default function SettingsSchoolPage() {
    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-8 p-8">
            <div>
                <h1 className="text-2xl font-semibold">School Settings</h1>
                <p className="text-muted-foreground mt-1">
                    Manage your school&apos;s configuration.
                </p>
            </div>

            <StreamsSection />
        </div>
    );
}
