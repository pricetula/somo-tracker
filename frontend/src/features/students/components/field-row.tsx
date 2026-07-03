/**
 * Field Row — label + value pair for profile displays.
 */

interface FieldRowProps {
    label: string;
    value: string;
}

export function FieldRow({ label, value }: FieldRowProps) {
    return (
        <div className="flex items-baseline gap-4 py-1.5">
            <span className="text-muted-foreground w-32 shrink-0 text-xs font-medium">{label}</span>
            <span className="text-sm">{value}</span>
        </div>
    );
}
