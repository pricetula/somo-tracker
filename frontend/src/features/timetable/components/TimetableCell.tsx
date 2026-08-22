"use client";

import { AlertCircle, Edit, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { TimetableCellView } from "../types";

interface TimetableCellProps {
    cell: TimetableCellView;
    onEdit?: (slotId: string) => void;
    onDelete?: (slotId: string) => void;
    onAdd?: (structureId: string) => void;
    isReadOnly?: boolean;
}

export function TimetableCell({
    cell,
    onEdit,
    onDelete,
    onAdd,
    isReadOnly = false,
}: TimetableCellProps) {
    const { structure, slot, hasConflict, conflictMessage } = cell;

    if (structure.is_break) {
        return (
            <div className="bg-muted/30 flex h-24 items-center justify-center rounded-md">
                <span className="text-muted-foreground text-sm font-medium">
                    {structure.period_name}
                </span>
            </div>
        );
    }

    if (!slot) {
        return (
            <div className="bg-background border-border relative h-24 rounded-md border border-dashed">
                {!isReadOnly && onAdd && (
                    <Button
                        variant="ghost"
                        size="icon"
                        className="absolute inset-0 flex items-center justify-center opacity-0 transition-opacity hover:opacity-100"
                        onClick={() => onAdd(structure.id)}
                        aria-label={`Add lesson to ${structure.period_name}`}
                    >
                        <PlusIcon className="text-muted-foreground h-6 w-6" />
                    </Button>
                )}
                <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                    <span className="text-muted-foreground/50 text-sm">Click to add lesson</span>
                </div>
            </div>
        );
    }

    return (
        <div
            className={`bg-background relative h-24 rounded-md border transition-colors ${
                hasConflict ? "border-destructive bg-destructive/5" : "border-border"
            }`}
        >
            <div className="space-y-1 p-2">
                <p className="text-foreground truncate text-xs font-medium">{slot.class_name}</p>
                {slot.learning_area_name && (
                    <p className="text-muted-foreground truncate text-xs">
                        {slot.learning_area_name}
                    </p>
                )}
                {slot.teacher_name && (
                    <p className="text-muted-foreground truncate text-xs">{slot.teacher_name}</p>
                )}
                {slot.room_identifier && (
                    <p className="text-muted-foreground truncate text-xs">
                        📍 {slot.room_identifier}
                    </p>
                )}
            </div>

            {hasConflict && conflictMessage && (
                <div className="absolute right-1 bottom-1 left-1">
                    <div className="text-destructive bg-destructive/10 flex items-center gap-1 rounded px-1.5 py-0.5 text-xs">
                        <AlertCircle className="h-3 w-3" />
                        <span className="truncate">{conflictMessage}</span>
                    </div>
                </div>
            )}

            {slot.session_status === "SUBMITTED" && (
                <div className="absolute top-1 right-1">
                    <span className="rounded-full bg-emerald-100 px-1.5 py-0.5 text-xs text-emerald-800">
                        Submitted
                    </span>
                </div>
            )}

            {slot.session_status === "SKIPPED" && (
                <div className="absolute top-1 right-1">
                    <span className="rounded-full bg-amber-100 px-1.5 py-0.5 text-xs text-amber-800">
                        Skipped
                    </span>
                </div>
            )}

            {!isReadOnly && (onEdit || onDelete) && (
                <DropdownMenu>
                    <DropdownMenuTrigger>
                        <Button
                            variant="ghost"
                            size="icon"
                            className="absolute top-1 right-1 opacity-0 transition-opacity group-hover:opacity-100 hover:opacity-100"
                        >
                            <MoreHorizontal className="h-4 w-4" />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="min-w-[160px]">
                        {onEdit && (
                            <DropdownMenuItem onClick={() => onEdit(slot.id)}>
                                <Edit className="mr-2 h-4 w-4" />
                                Edit
                            </DropdownMenuItem>
                        )}
                        {onDelete && (
                            <DropdownMenuItem
                                onClick={() => onDelete(slot.id)}
                                className="text-destructive focus:text-destructive"
                            >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete
                            </DropdownMenuItem>
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>
            )}
        </div>
    );
}

function PlusIcon({ className }: { className?: string }) {
    return (
        <svg
            className={className}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
        >
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
        </svg>
    );
}

function MoreHorizontal({ className }: { className?: string }) {
    return (
        <svg
            className={className}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
        >
            <circle cx="12" cy="12" r="1" />
            <circle cx="19" cy="12" r="1" />
            <circle cx="5" cy="12" r="1" />
        </svg>
    );
}
