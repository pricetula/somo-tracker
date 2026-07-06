import { Button } from "@/components/ui/button";

interface StudentsImportSelectorProps {
    onSelect: (type: "manual" | "file") => void;
    isDialogVersion: boolean;
}

export function StudentsImportSelector({ onSelect, isDialogVersion }: StudentsImportSelectorProps) {
    return (
        <article>
            {!isDialogVersion && <h1>Add Students</h1>}
            <p>How would you like to add students?</p>
            <ul>
                <li>
                    <Button onClick={() => onSelect("manual")}>Manual Entry</Button>
                </li>
                <li>
                    <Button onClick={() => onSelect("file")}>Import File</Button>
                </li>
            </ul>
        </article>
    );
}
