import { Button } from "@/components/ui/button";

interface StudentManualImportFormProps {
    onReset: () => void;
}

export function StudentManualImportForm({ onReset }: StudentManualImportFormProps) {
    return (
        <section>
            <Button variant="destructive" onClick={onReset}>
                Reset
            </Button>
        </section>
    );
}
