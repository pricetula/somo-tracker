"use client";
import { useSidebar } from "@/components/ui/sidebar";

interface AppLayoutProps {
    children: React.ReactNode;
}

export function ChildrenWrapper({ children }: AppLayoutProps) {
    const { state, isMobile } = useSidebar();

    const sidebarWidth = isMobile
        ? "0px"
        : state === "collapsed"
          ? "var(--sidebar-width-icon)"
          : "var(--sidebar-width)";

    return (
        <div
            className="max-w-7xl overflow-x-hidden px-6 py-2 transition-[width] duration-200 ease-linear"
            style={{ width: `calc(100vw - ${sidebarWidth})` }}
        >
            {children}
        </div>
    );
}
