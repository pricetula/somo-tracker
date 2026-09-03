/**
 * Add Class Page — Full page render for /classes/add.
 *
 * On hard refresh, renders the add class form as a standalone page.
 * When client-navigated from the classes table, it is intercepted
 * by @modal/(.)classes/add and rendered as a dialog overlay.
 */

import { AddClassForm } from "@/features/classes";

export default function AddClassPage() {
    return (
        <div className="mx-auto max-w-lg p-6">
            <h1 className="mb-1 text-lg font-semibold">Create Class</h1>
            <p className="text-muted-foreground mb-6">
                Create a new class by selecting a grade level and stream. The current academic year
                and term will be used automatically.
            </p>
            <AddClassForm />
        </div>
    );
}
