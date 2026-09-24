"use client";

import React from "react";
import { HelpCircle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface HelpProps {
    children: React.ReactNode;
}

/**
 * Help — Inline help tooltip for charts and visualisations.
 *
 * Renders a small help icon that shows descriptive text on hover.
 * Usage: inline inside a heading or label, wrapping the explanation.
 */
export function Help({ children }: HelpProps) {
    return (
        <Tooltip>
            <TooltipTrigger>
                <span className="text-muted-foreground hover:text-foreground ml-1 inline-flex cursor-help items-center align-middle transition-colors">
                    <HelpCircle className="h-3 w-3" />
                </span>
            </TooltipTrigger>
            <TooltipContent className="side-top max-w-xs p-3 text-xs">{children}</TooltipContent>
        </Tooltip>
    );
}
