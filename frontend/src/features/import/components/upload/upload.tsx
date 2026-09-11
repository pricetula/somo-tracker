import { Button } from "@/components/ui/button";
import { X } from "lucide-react";

interface UploadProps {
    onCancel: () => void;
}

export function Upload({ onCancel }: UploadProps) {
    return (
        <div>
            <Button size="icon" variant="outline" onClick={onCancel}>
                <X />
            </Button>
            Upload
        </div>
    );
}
