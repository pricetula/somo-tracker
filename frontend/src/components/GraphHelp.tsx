"use client";

import React from "react";
import { HelpCircle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface GraphHelpProps {
    children: React.ReactNode;
}

/**
 * GraphHelp — Inline help tooltip for charts and visualisations.
 *
 * Renders a small help icon that shows descriptive text on hover.
 * Usage: inline inside a heading or label, wrapping the explanation.
 */
export function GraphHelp({ children }: GraphHelpProps) {
    return (
        <Tooltip>
            <TooltipTrigger>
                <span className="text-muted-foreground hover:text-foreground ml-1 inline-flex cursor-help items-center align-middle transition-colors">
                    <HelpCircle className="h-4 w-4" />
                </span>
            </TooltipTrigger>
            <TooltipContent className="side-top max-w-xs p-3 text-xs">{children}</TooltipContent>
        </Tooltip>
    );
}
