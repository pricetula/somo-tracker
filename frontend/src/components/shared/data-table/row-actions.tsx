/**
 * RowActions — reusable dropdown menu for per-row actions.
 *
 * Renders a shadcn DropdownMenu with an actions trigger and
 * AlertDialog confirmation for destructive actions.
 *
 * Usage:
 *   <RowActions
 *     rowId={row.id}
 *     label={row.name}
 *     onDelete={() => deleteMutation.mutate(row.id)}
 *   />
 *
 * For custom non-delete actions:
 *   <RowActions
 *     rowId={row.id}
 *     label={row.name}
 *     actions={[
 *       { label: "Cancel", icon: XCircle, onClick: () => cancelMutation.mutate(row.id) },
 *       { label: "Waive", icon: FileX, onClick: () => waiveMutation.mutate(row.id) },
 *     ]}
 *   />
 */

"use client";

import { MoreHorizontal, Trash2, type LucideIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from "@/components/ui/alert-dialog";

export interface RowAction {
    label: string;
    icon: LucideIcon;
    /** If true, shows delete-red styling and requires confirmation dialog. */
    destructive?: boolean;
    /** Custom confirmation title, defaults to label. */
    confirmTitle?: string;
    /** Custom confirmation description. */
    confirmDescription?: string;
    onClick: () => void;
}

interface RowActionsProps {
    /** Unique identifier for the row. */
    rowId: string;
    /** Human-readable label shown in confirmation dialogs (e.g. the item name). */
    label?: string;
    /** When provided, renders a "Delete" action with confirmation dialog. */
    onDelete?: () => void;
    /** Additional custom actions shown above the delete action. */
    actions?: RowAction[];
    /** When true, disables all actions (e.g. while an operation is in progress). */
    disabled?: boolean;
}

export function RowActions({ rowId, label, onDelete, actions, disabled }: RowActionsProps) {
    const menuItems = actions ?? [];
    const destructiveItem = menuItems.find((a) => a.destructive);

    return (
        <div className="flex items-center justify-end">
            <AlertDialog>
                <DropdownMenu>
                    <DropdownMenuTrigger
                        render={
                            <Button
                                variant="ghost"
                                size="icon-sm"
                                disabled={disabled}
                                className="text-muted-foreground data-[state=open]:bg-muted size-6"
                            >
                                <MoreHorizontal className="size-3.5" />
                                <span className="sr-only">Actions for {label ?? rowId}</span>
                            </Button>
                        }
                    />
                    <DropdownMenuContent align="end" sideOffset={4}>
                        {menuItems.map((action) => {
                            const Icon = action.icon;
                            return (
                                <DropdownMenuItem
                                    key={action.label}
                                    className={
                                        action.destructive
                                            ? "text-destructive cursor-pointer"
                                            : "cursor-pointer"
                                    }
                                    onClick={() => {
                                        if (action.onClick) {
                                            action.onClick();
                                        }
                                    }}
                                >
                                    <Icon className="mr-2 size-3.5" />
                                    {action.label}
                                </DropdownMenuItem>
                            );
                        })}
                        {onDelete && (
                            <AlertDialogTrigger
                                render={
                                    <DropdownMenuItem
                                        className="text-destructive cursor-pointer"
                                        onSelect={(e) => e.preventDefault()}
                                    >
                                        <Trash2 className="mr-2 size-3.5" />
                                        Delete
                                    </DropdownMenuItem>
                                }
                            />
                        )}
                    </DropdownMenuContent>
                </DropdownMenu>

                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>
                            {destructiveItem?.confirmTitle ?? `Delete ${label ?? "item"}`}
                        </AlertDialogTitle>
                        <AlertDialogDescription>
                            {destructiveItem?.confirmDescription ??
                                (label
                                    ? `Are you sure you want to delete "${label}"? This action cannot be undone.`
                                    : "Are you sure you want to delete this item? This action cannot be undone.")}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                            variant="destructive"
                            onClick={(e) => {
                                e.preventDefault();
                                if (destructiveItem) {
                                    destructiveItem.onClick();
                                } else {
                                    onDelete?.();
                                }
                            }}
                        >
                            {destructiveItem?.label ?? "Delete"}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
