"use client";

import type { FilterGroup, FilterItem } from "./types";
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
import { Filter } from "lucide-react";
import { cn } from "@/lib/utils";

// ─── Helper: render filter items recursively ─────────────────────────────

function renderFilterItems(
    items: FilterItem[],
    groupId: string,
    type: "single" | "multi",
    activeValues: string[],
    onToggleValue: (groupId: string, value: string) => void,
    onSelectSingle: (groupId: string, value: string) => void
) {
    return items.map((item) => {
        if (item.submenu && item.submenu.length > 0) {
            return (
                <DropdownMenuSub key={item.id}>
                    <DropdownMenuSubTrigger>
                        {item.icon && <item.icon className="size-3.5" />}
                        {item.label}
                    </DropdownMenuSubTrigger>
                    <DropdownMenuSubContent>
                        {renderFilterItems(
                            item.submenu,
                            groupId,
                            type,
                            activeValues,
                            onToggleValue,
                            onSelectSingle
                        )}
                    </DropdownMenuSubContent>
                </DropdownMenuSub>
            );
        }

        const isActive = activeValues.includes(item.value);

        if (type === "multi") {
            return (
                <DropdownMenuCheckboxItem
                    key={item.id}
                    checked={isActive}
                    onSelect={(e) => {
                        e.preventDefault();
                        onToggleValue(groupId, item.value);
                    }}
                >
                    {item.icon && <item.icon className="size-3.5" />}
                    {item.label}
                </DropdownMenuCheckboxItem>
            );
        }

        return (
            <DropdownMenuRadioItem
                key={item.id}
                value={item.value}
                onSelect={() => onSelectSingle(groupId, item.value)}
            >
                {item.icon && <item.icon className="size-3.5" />}
                {item.label}
            </DropdownMenuRadioItem>
        );
    });
}

// ─── Props ───────────────────────────────────────────────────────────────

interface FilterDropdownProps {
    groups: FilterGroup[];
    activeFilters: Record<string, string[]>;
    onToggleValue: (groupId: string, value: string) => void;
    onSelectSingle: (groupId: string, value: string) => void;
    disabled?: boolean;
}

// ─── Component ───────────────────────────────────────────────────────────

export function FilterDropdown({
    groups,
    activeFilters,
    onToggleValue,
    onSelectSingle,
    disabled,
}: FilterDropdownProps) {
    const hasActiveFilters = Object.values(activeFilters).some((vals) => vals.length > 0);

    return (
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
                                {group.icon && <group.icon className="mr-1.5 inline size-3.5" />}
                                {group.label}
                            </DropdownMenuLabel>
                            {group.type === "single" ? (
                                <DropdownMenuRadioGroup value={activeFilters[group.id]?.[0] ?? ""}>
                                    {renderFilterItems(
                                        group.items,
                                        group.id,
                                        group.type,
                                        activeFilters[group.id] ?? [],
                                        onToggleValue,
                                        onSelectSingle
                                    )}
                                </DropdownMenuRadioGroup>
                            ) : (
                                renderFilterItems(
                                    group.items,
                                    group.id,
                                    group.type,
                                    activeFilters[group.id] ?? [],
                                    onToggleValue,
                                    onSelectSingle
                                )
                            )}
                        </DropdownMenuGroup>
                    </div>
                ))}
            </DropdownMenuContent>
        </DropdownMenu>
    );
}
