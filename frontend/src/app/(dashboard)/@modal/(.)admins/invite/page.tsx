/**
 * Modal route — renders ImportOrchestrator wrapped in a Dialog shell.
 * Matches the intercepting route `@modal/(.)admins/invite`.
 */

import { ImportOrchestrator } from "@/features/import";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

export default function ImportModalPage() {
    return (
        <Dialog open>
            <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
                <DialogHeader>
                    <DialogTitle>Import Data</DialogTitle>
                </DialogHeader>
                <ImportOrchestrator />
            </DialogContent>
        </Dialog>
    );
}
