import { ThemeSwitch } from "../../../components/ui/theme-switch";

export default function SettingsPage() {
    return (
        <section className="space-y-6">
            <h1 className="text-2xl font-bold">Settings</h1>
            <div className="rounded-lg border p-6">
                <h2 className="mb-4 text-lg font-semibold">Appearance</h2>
                <ThemeSwitch />
            </div>
        </section>
    );
}
