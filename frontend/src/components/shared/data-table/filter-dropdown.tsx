"use client";

import type { ReactNode } from "react";
import type { FilterGroup, FilterItem, SubFilterItem } from "./types";
import { Button } from "@/components/ui/button";
import {
    DropdownMenu,
    DropdownMenuCheckboxItem,
    DropdownMenuContent,
    DropdownMenuGroup,
    DropdownMenuLabel,
    DropdownMenuRadioGroup,
    DropdownMenuRadioItem,
    DropdownMenuSeparator,
    DropdownMenuSub,
    DropdownMenuSubContent,
    DropdownMenuSubTrigger,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Filter, X } from "lucide-react";
import { cn } from "@/lib/utils";

// ─── Props ───────────────────────────────────────────────────────────────

interface FilterDropdownProps {
    groups: FilterGroup[];
    /**
     * Active filter values keyed by **item id** (not group id).
     *
     * - `"button"` items: store the item's own `value` string when active.
     * - `"sub_menu_single"` items: store the selected `SubFilterItem.value` when one is chosen.
     * - `"sub_menu_multi"` items: store an array of selected `SubFilterItem.values`.
     */
    activeFilters: Record<string, string | string[]>;
    /**
     * Called when a `"button"` item is toggled. Receives the item id.
     * The consumer toggles that item's `value` on/off in activeFilters.
     */
    onToggleButton: (itemId: string, itemValue: string) => void;
    /**
     * Called when a `"sub_menu_single"` sub-item is selected.
     * Receives the parent item id and the selected sub-item value.
     * Passing the same value again deselects it (radio toggle-off).
     */
    onSelectSingle: (itemId: string, subValue: string) => void;
    /**
     * Called when a `"sub_menu_multi"` sub-item is toggled.
     * Receives the parent item id and the toggled sub-item value.
     */
    onToggleMulti: (itemId: string, subValue: string) => void;
    /**
     * Called when an active filter pill's remove button is clicked.
     *
     * - `"button"` items: `subValue` is omitted — the entire filter is removed.
     * - `"sub_menu_single"` items: `subValue` is omitted — the entire filter is removed.
     * - `"sub_menu_multi"` items: `subValue` is the specific sub-item value to remove.
     */
    onRemoveFilter: (itemId: string, subValue?: string) => void;
    disabled?: boolean;
}

// ─── Helpers ─────────────────────────────────────────────────────────────

function isMulti(subFilterValue: unknown): subFilterValue is string[] {
    return Array.isArray(subFilterValue);
}

// ─── Active filter pills ────────────────────────────────────────────────

interface ActiveFilterPill {
    itemId: string;
    label: ReactNode;
    subValue?: string;
}

function resolveActivePills(
    groups: FilterGroup[],
    activeFilters: Record<string, string | string[]>
): ActiveFilterPill[] {
    const pills: ActiveFilterPill[] = [];

    for (const group of groups) {
        for (const item of group.items) {
            const val = activeFilters[item.id];
            if (val === undefined || val === "") continue;

            if (item.type === "button") {
                pills.push({ itemId: item.id, label: item.label });
            } else if (item.type === "sub_menu_single" && typeof val === "string") {
                const sub = item.submenu.find((s) => s.value === val);
                if (sub) {
                    pills.push({ itemId: item.id, label: sub.label, subValue: sub.value });
                }
            } else if (item.type === "sub_menu_multi" && isMulti(val)) {
                for (const v of val) {
                    const sub = item.submenu.find((s) => s.value === v);
                    if (sub) {
                        pills.push({ itemId: item.id, label: sub.label, subValue: sub.value });
                    }
                }
            }
        }
    }

    return pills;
}

// ─── Render sub-items ────────────────────────────────────────────────────

function renderSubItems(
    parentItem: FilterItem & { submenu: SubFilterItem[] },
    activeFilters: Record<string, string | string[]>,
    onSelectSingle: (itemId: string, subValue: string) => void,
    onToggleMulti: (itemId: string, subValue: string) => void
) {
    const parentId = parentItem.id;

    if (parentItem.type === "sub_menu_single") {
        const currentValue = (activeFilters[parentId] as string | undefined) ?? "";

        return (
            <DropdownMenuRadioGroup
                value={currentValue}
                onValueChange={(value) => onSelectSingle(parentId, value)}
            >
                {parentItem.submenu.map((sub) => (
                    <DropdownMenuRadioItem key={sub.id} value={sub.value}>
                        {sub.icon && <sub.icon className="size-3.5" />}
                        {sub.label}
                    </DropdownMenuRadioItem>
                ))}
            </DropdownMenuRadioGroup>
        );
    }

    // sub_menu_multi
    const activeValues = isMulti(activeFilters[parentId])
        ? (activeFilters[parentId] as string[])
        : [];

    return parentItem.submenu.map((sub) => {
        const isChecked = activeValues.includes(sub.value);

        return (
            <DropdownMenuCheckboxItem
                key={sub.id}
                checked={isChecked}
                onSelect={(e) => {
                    e.preventDefault();
                    onToggleMulti(parentId, sub.value);
                }}
            >
                {sub.icon && <sub.icon className="size-3.5" />}
                {sub.label}
            </DropdownMenuCheckboxItem>
        );
    });
}

// ─── Render items (section contents, not wrapped in group) ───────────────

function renderItems(
    items: FilterItem[],
    activeFilters: Record<string, string | string[]>,
    onToggleButton: (itemId: string, itemValue: string) => void,
    onSelectSingle: (itemId: string, subValue: string) => void,
    onToggleMulti: (itemId: string, subValue: string) => void
) {
    return items.map((item) => {
        // ── Button item: toggle directly ────────────────────────────
        if (item.type === "button") {
            const isActive =
                typeof activeFilters[item.id] === "string" && activeFilters[item.id] !== "";

            return (
                <DropdownMenuCheckboxItem
                    key={item.id}
                    checked={isActive}
                    onSelect={(e) => {
                        e.preventDefault();
                        onToggleButton(item.id, item.value);
                    }}
                >
                    {item.icon && <item.icon className="size-3.5" />}
                    {item.label}
                </DropdownMenuCheckboxItem>
            );
        }

        // ── Sub-menu item: single or multi ──────────────────────────
        return (
            <DropdownMenuSub key={item.id}>
                <DropdownMenuSubTrigger>
                    {item.icon && <item.icon className="size-3.5" />}
                    {item.label}
                </DropdownMenuSubTrigger>
                <DropdownMenuSubContent>
                    {renderSubItems(item, activeFilters, onSelectSingle, onToggleMulti)}
                </DropdownMenuSubContent>
            </DropdownMenuSub>
        );
    });
}

// ─── Component ───────────────────────────────────────────────────────────

export function FilterDropdown({
    groups,
    activeFilters,
    onToggleButton,
    onSelectSingle,
    onToggleMulti,
    onRemoveFilter,
    disabled,
}: FilterDropdownProps) {
    const hasActiveFilters = Object.values(activeFilters).some((val) => {
        if (Array.isArray(val)) return val.length > 0;
        return val !== "";
    });

    const activePills = resolveActivePills(groups, activeFilters);

    return (
        <>
            <DropdownMenu>
                <DropdownMenuTrigger asChild disabled={disabled}>
                    <Button
                        variant="outline"
                        size="icon"
                        className={cn(hasActiveFilters && "bg-muted")}
                    >
                        <Filter className="size-3.5" />
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                    {groups.map((group, idx) => (
                        <div key={group.id}>
                            {idx > 0 && <DropdownMenuSeparator />}
                            <DropdownMenuGroup>
                                <DropdownMenuLabel>
                                    {group.icon && (
                                        <group.icon className="mr-1.5 inline size-3.5" />
                                    )}
                                    {group.label}
                                </DropdownMenuLabel>
                                {renderItems(
                                    group.items,
                                    activeFilters,
                                    onToggleButton,
                                    onSelectSingle,
                                    onToggleMulti
                                )}
                            </DropdownMenuGroup>
                        </div>
                    ))}
                </DropdownMenuContent>
            </DropdownMenu>

            {activePills.length > 0 && (
                <div className="flex flex-wrap items-center gap-1.5">
                    {activePills.map((pill) => (
                        <span
                            key={`${pill.itemId}${pill.subValue ? `:${pill.subValue}` : ""}`}
                            className="bg-muted text-foreground inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium"
                        >
                            {pill.label}
                            <button
                                type="button"
                                onClick={() => onRemoveFilter(pill.itemId, pill.subValue)}
                                className="text-muted-foreground hover:text-foreground rounded-sm transition-colors"
                            >
                                <X className="size-3" />
                            </button>
                        </span>
                    ))}
                </div>
            )}
        </>
    );
}
