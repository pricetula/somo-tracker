/**
 * GraphHelp — Click-triggered popover with a quick explanation of what a graph describes.
 *
 * Usage:
 *   <GraphHelp>
 *     Line chart showing attendance trends across multiple terms.
 *   </GraphHelp>
 *
 * Place it inline next to the graph title.
 */
"use client";

import { HelpCircle } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface GraphHelpProps {
    children: React.ReactNode;
}

export function GraphHelp({ children }: GraphHelpProps) {
    return (
        <Popover>
            <PopoverTrigger asChild>
                <span className="text-muted-foreground hover:text-foreground ml-1 inline-flex cursor-help items-center align-middle transition-colors">
                    <HelpCircle className="h-3.5 w-3.5" />
                </span>
            </PopoverTrigger>
            <PopoverContent className="max-w-xs text-xs leading-relaxed" side="top" align="center">
                {children}
            </PopoverContent>
        </Popover>
    );
}
