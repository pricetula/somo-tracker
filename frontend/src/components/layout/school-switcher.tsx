import React from "react";

export function SchoolSwitcher() {
    return (
        <div className="flex items-center gap-2 px-2 py-1">
            <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-6 items-center justify-center rounded-lg text-xs font-medium">
                S
            </div>
            <div className="grid flex-1 text-left leading-tight">
                <span className="truncate font-medium">School name</span>
            </div>
        </div>
    );
}
