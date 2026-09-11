import { Button } from "@/components/ui/button";
import { X } from "lucide-react";

interface ManualImportProps {
    onCancel: () => void;
}

export function ManualImport({ onCancel }: ManualImportProps) {
    return (
        <div>
            <Button size="icon" variant="outline" onClick={onCancel}>
                <X />
            </Button>
            Manual import
        </div>
    );
}
